import type { TypesMacroInfo as MacroInfo } from '@/api'


export interface ApplyDialogProps {
  macro: MacroInfo
  onClose: () => void
  onApplied: () => void
}
