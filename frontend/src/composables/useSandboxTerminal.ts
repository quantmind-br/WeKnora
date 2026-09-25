import { onUnmounted, ref, type Ref } from 'vue'
import { post } from '@/utils/request'
import { readStoredPtyId, writeStoredPtyId } from '@/utils/sandboxPtyId'

export type SandboxTerminalStatus =
  /** The session sandbox was found paused; waking it requires user confirmation. */
  | 'paused'
  | 'connecting'
  | 'ready'
  /** The session was found to have no running sandbox; creating one requires user confirmation. */
  | 'needs_provision'
  | 'exited'
  /** The user confirmed creation, but the backend has nowhere to create it (the agent has no sandbox backend configured). */
  | 'no_sandbox'
  | 'unsupported'
  | 'idle'
  | 'unauthorized'
  | 'error'

export type SandboxTerminalControlFrame = {
  type: string
  code?: string
  message?: string
  pty_id?: number
  backend?: string
  exit_code?: number | null
  cols?: number
  rows?: number
}

const RECONNECT_BASE_DELAY_MS = 1000
const RECONNECT_MAX_DELAY_MS = 30000
const APP_PING_INTERVAL_MS = 25000
const PENDING_OUTPUT_MAX_BYTES = 1024 * 1024
// The server caps a single input frame at 4 KiB (terminalMaxInputBytes); when gorilla exceeds it,
// it closes the connection outright, and xterm delivers a paste to onData in one piece, so pasting a script drops the line.
// Slice by bytes before sending: the PTY is a byte stream, so a cut in the middle of a multi-byte character is reassembled in order as-is.
const INPUT_FRAME_MAX_BYTES = 2048

function resolveWsBase(): string {
  const base = (import.meta.env.BASE_URL || '/').replace(/\/+$/, '')
  const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
  return `${protocol}://${window.location.host}${base}`
}

async function mintTerminalTicket(sessionId: string): Promise<string> {
  const res = await post<{ success?: boolean; data?: { ticket?: string } }>(
    `/api/v1/sessions/${encodeURIComponent(sessionId)}/sandbox/terminal-ticket`,
    {},
  )
  const ticket = res?.data?.ticket
  if (!ticket) {
    throw new Error('missing terminal ticket')
  }
  return ticket
}

/**
  * provisionAttempted decides what SANDBOX_NOT_BOUND means: without creation intent it only means "there is
  * no sandbox yet, do you want one"; only when creation was requested and still failed is there truly nowhere to create it.
 */
function statusFromErrorCode(
  code: string | undefined,
  provisionAttempted: boolean,
): SandboxTerminalStatus {
  if (code === 'SANDBOX_NOT_BOUND') {
    return provisionAttempted ? 'no_sandbox' : 'needs_provision'
  }
  if (code === 'SANDBOX_PAUSED') return 'paused'
  if (code === 'TERMINAL_UNSUPPORTED') return 'unsupported'
  if (code === 'IDLE_DISCONNECTED') return 'idle'
  if (code === 'AUTH_REVOKED') return 'unauthorized'
  return 'error'
}

export type SandboxTerminalSession = {
  status: Ref<SandboxTerminalStatus>
  /**
    * Injected by SandboxTerminal.vue: writes PTY output into xterm.
    * Binary frames that arrive before the handler is registered are queued first, so the bash prompt is not
    * dropped before xterm mounts. Pass null on unmount to start buffering again.
   */
  onOutput: (handler: ((data: Uint8Array) => void) | null) => void
  /**
    * Connect and open the terminal; idempotent while a connection is already in progress.
   *
    * provision means "this is an explicit user action; the backend may create or wake the sandbox". Only a button
    * click should pass true: creating/waking a sandbox is real, billable infrastructure. Auto-connect when the panel
    * opens never passes it: a running sandbox is attached directly, while a paused or not-yet-created one falls back to asking the user.
   */
  connect: (options?: { provision?: boolean; cols?: number; rows?: number }) => void
  /** Send keyboard input (the raw string from xterm onData). */
  sendInput: (data: string) => void
  /** Sync the terminal size; call after ready. */
  resize: (cols: number, rows: number) => void
  /** Disconnect and stop reconnecting (call on component unmount / session switch). */
  dispose: () => void
}

/**
  * All three arguments must be live refs (`toRef(props, …)`), not snapshots like
  * `ref(props.x)`: every openSocket re-reads them, so after the user switches agent (including the shared source space),
  * the next connection creates the sandbox with the new agent's sandbox config.
 */
export function useSandboxTerminal(
  sessionId: Ref<string>,
  agentId: Ref<string | undefined>,
  agentSourceTenantId: Ref<string | number | null | undefined> = ref(undefined),
): SandboxTerminalSession {
  const status = ref<SandboxTerminalStatus>('connecting')

  let ws: WebSocket | null = null
  let opening = false
  let outputHandler: ((data: Uint8Array) => void) | null = null
  let pendingOutput: Uint8Array[] = []
  let pendingOutputBytes = 0
  let disposed = false
  let reconnectAttempt = 0
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let pingTimer: ReturnType<typeof setInterval> | null = null
  let pendingResize: { cols: number; rows: number } | null = null
  let pendingGeometry: { cols: number; rows: number } | null = null
  // A refresh / closing the panel tears down this closure; the PID is kept in sessionStorage so a reattach can send pty_id.
  let lastPid: number | null = readStoredPtyId(sessionId.value)
  // Whether the current connection carries creation intent. Only connect({ provision: true }) sets it to true.
  let allowProvision = false

  const textEncoder = new TextEncoder()

  function rememberPid(next: number | null) {
    lastPid = next
    writeStoredPtyId(sessionId.value, next)
  }

  function clearReconnectTimer() {
    if (reconnectTimer !== null) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
  }

  function clearPingTimer() {
    if (pingTimer !== null) {
      clearInterval(pingTimer)
      pingTimer = null
    }
  }

  function scheduleReconnect() {
    if (disposed) return
    clearReconnectTimer()
    // Auto-reconnect must not revive a reclaimed sandbox: only a user click carries creation intent. If the
    // sandbox is reclaimed while disconnected, the reconnect gets needs_provision and the user decides whether to rebuild it.
    allowProvision = false
    const delay = Math.min(
      RECONNECT_BASE_DELAY_MS * 2 ** reconnectAttempt,
      RECONNECT_MAX_DELAY_MS,
    )
    reconnectAttempt += 1
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      void openSocket()
    }, delay)
  }

  async function openSocket() {
    if (disposed || opening || ws) return
    const sid = sessionId.value
    if (!sid) return

    opening = true
    status.value = 'connecting'
    try {
      const ticket = await mintTerminalTicket(sid)
      if (disposed) return
      const query = new URLSearchParams({ ticket })
      if (allowProvision) {
        query.set('provision', '1')
        // agent_id only matters when creation is allowed: it is the config source used to create the sandbox the first time.
        const agent = agentId.value
        if (agent && agent !== 'builtin-quick-answer') {
          query.set('agent_id', agent)
        }
        const sourceTenant = agentSourceTenantId.value
        if (sourceTenant != null && String(sourceTenant).trim() !== '') {
          query.set('agent_source_tenant_id', String(sourceTenant).trim())
        }
      }
      if (lastPid && lastPid > 0) {
        query.set('pty_id', String(lastPid))
      }
      if (pendingGeometry) {
        query.set('cols', String(pendingGeometry.cols))
        query.set('rows', String(pendingGeometry.rows))
      }
      const socket = new WebSocket(
        `${resolveWsBase()}/api/v1/sessions/${encodeURIComponent(sid)}/sandbox/terminal?${query.toString()}`,
      )
      if (disposed) {
        socket.close()
        return
      }
      ws = socket
    } catch {
      ws = null
      if (disposed) return
      status.value = 'error'
      scheduleReconnect()
      return
    } finally {
      opening = false
    }
    if (!ws) return
    ws.binaryType = 'arraybuffer'

    ws.onopen = () => {
      clearPingTimer()
      pingTimer = setInterval(() => {
        if (ws && ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: 'ping' }))
        }
      }, APP_PING_INTERVAL_MS)
    }

    ws.onmessage = (event) => {
      if (typeof event.data === 'string') {
        handleControlFrame(event.data)
        return
      }
      const data =
        event.data instanceof ArrayBuffer ? new Uint8Array(event.data) : new Uint8Array(0)
      deliverOutput(data)
    }

    ws.onclose = (event) => {
      clearPingTimer()
      ws = null
      if (disposed) return
      const reason = typeof event.reason === 'string' ? event.reason : ''
      if (
        event.code === 1008
        || reason === 'SANDBOX_NOT_BOUND'
        || reason === 'SANDBOX_PAUSED'
        || reason === 'TERMINAL_UNSUPPORTED'
        || reason === 'IDLE_DISCONNECTED'
        || reason === 'AUTH_REVOKED'
      ) {
        if (reason) {
          status.value = statusFromErrorCode(reason, allowProvision)
        } else if (status.value === 'ready' || status.value === 'connecting') {
          status.value = 'error'
        }
        // idle / unauthorized / paused: the shell is still in the sandbox, so the PID must be kept to reattach.
        // Sandbox gone or backend unsupported: the PID is no longer valid, so clear it to avoid the next Create hitting a dead process.
        if (status.value !== 'idle' && status.value !== 'unauthorized' && status.value !== 'paused') {
          rememberPid(null)
        }
        return
      }
      if (
        status.value !== 'needs_provision'
        && status.value !== 'paused'
        && status.value !== 'no_sandbox'
        && status.value !== 'unsupported'
        && status.value !== 'exited'
        && status.value !== 'idle'
        && status.value !== 'unauthorized'
      ) {
        status.value = 'error'
        scheduleReconnect()
      }
    }

    ws.onerror = () => {
      // onclose fires right after; everything is handled there.
    }
  }

  function handleControlFrame(raw: string) {
    let frame: SandboxTerminalControlFrame
    try {
      frame = JSON.parse(raw)
    } catch {
      return
    }
    switch (frame.type) {
      case 'ready':
        status.value = 'ready'
        rememberPid(typeof frame.pty_id === 'number' ? frame.pty_id : null)
        reconnectAttempt = 0
        // Creation intent ends here. It only serves to interpret SANDBOX_NOT_BOUND: before connecting it means "there is
        // no sandbox yet, do you want one"; received after connecting, it always means "the sandbox was reclaimed". Not resetting
        // would misread a reclaim as no_sandbox ("the agent has no backend configured"), a message unrelated to the real cause.
        allowProvision = false
        if (pendingResize) {
          sendResize(pendingResize.cols, pendingResize.rows)
          pendingResize = null
        }
        break
      case 'exited': {
        status.value = 'exited'
        rememberPid(null)
        break
      }
      case 'error': {
        status.value = statusFromErrorCode(frame.code, allowProvision)
        if (status.value !== 'idle' && status.value !== 'unauthorized' && status.value !== 'paused') {
          rememberPid(null)
        }
        break
      }
      default:
        break
    }
  }

  function deliverOutput(data: Uint8Array) {
    if (data.length === 0) return
    if (outputHandler) {
      outputHandler(data)
      return
    }
    pendingOutput.push(data)
    pendingOutputBytes += data.length
    while (pendingOutputBytes > PENDING_OUTPUT_MAX_BYTES && pendingOutput.length > 0) {
      const dropped = pendingOutput.shift()
      if (dropped) pendingOutputBytes -= dropped.length
    }
  }

  function sendRaw(frame: SandboxTerminalControlFrame) {
    ws?.send(JSON.stringify(frame))
  }

  function sendResize(cols: number, rows: number) {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      pendingResize = { cols, rows }
      return
    }
    sendRaw({ type: 'resize', cols, rows })
  }

  const session: SandboxTerminalSession = {
    status,
    onOutput(handler) {
      outputHandler = handler
      if (!handler) return
      const queued = pendingOutput
      pendingOutput = []
      pendingOutputBytes = 0
      for (const chunk of queued) {
        handler(chunk)
      }
    },
    connect(options) {
      if (disposed || ws || opening) return
      // Drop any pending auto-reconnect, otherwise it would dial again later with provision=false
      // and race the user's click for the connection.
      clearReconnectTimer()
      allowProvision = options?.provision === true
      reconnectAttempt = 0
      const cols = options?.cols
      const rows = options?.rows
      pendingGeometry =
        cols && rows && cols > 0 && rows > 0 ? { cols, rows } : null
      void openSocket()
    },
    sendInput(data) {
      if (!ws || ws.readyState !== WebSocket.OPEN) return
      const bytes = textEncoder.encode(data)
      for (let offset = 0; offset < bytes.length; offset += INPUT_FRAME_MAX_BYTES) {
        ws.send(bytes.subarray(offset, offset + INPUT_FRAME_MAX_BYTES))
      }
    },
    resize: sendResize,
    dispose() {
      disposed = true
      clearReconnectTimer()
      clearPingTimer()
      if (ws) {
        const socket = ws
        ws = null
        socket.onclose = null
        socket.onerror = null
        socket.onmessage = null
        socket.close()
      }
    },
  }

  onUnmounted(() => session.dispose())

  return session
}
