export type GuidePlacement = 'right' | 'left' | 'bottom' | 'top'

export interface SpotlightGuideStep {
  key: string
  /** CSS selector for the highlight target; omitted means a centered card */
  target?: string
  placement?: GuidePlacement
  before?: () => void | Promise<void>
  /** Whether to skip this step when the target doesn't exist */
  optional?: boolean
  /** When true, prompts the user to click the highlighted area directly, without showing "Next" */
  interact?: boolean
}
