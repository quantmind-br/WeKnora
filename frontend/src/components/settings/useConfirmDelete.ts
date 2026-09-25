import { DialogPlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'

interface ConfirmDeleteOptions {
  title?: string
  body: string
  confirmText?: string
  cancelText?: string
  onConfirm: () => Promise<void> | void
}

/**
 * Unified delete confirmation interaction, based on TDesign DialogPlugin.confirm.
 * Replaces window.confirm / t-popconfirm / custom Dialog implementations scattered throughout the codebase.
 * Destructive actions consistently use the danger theme and a red confirm button; only lightweight inline actions use t-popconfirm.
 */
export function useConfirmDelete() {
  const { t } = useI18n()

  return (opts: ConfirmDeleteOptions) => {
    const dialog = DialogPlugin.confirm({
      header: opts.title || (t('common.confirmDelete') as string),
      body: opts.body,
      confirmBtn: { content: opts.confirmText || (t('common.delete') as string), theme: 'danger' },
      cancelBtn: opts.cancelText || (t('common.cancel') as string),
      theme: 'danger',
      onConfirm: async () => {
        try {
          await opts.onConfirm()
        } finally {
          dialog.hide()
        }
      }
    })
    return dialog
  }
}
