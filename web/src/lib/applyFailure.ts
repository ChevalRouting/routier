import type { TypesApplyResult, TypesArtifactError } from '@/api'

type ErrShape = { message?: string; body?: BodyShape }

type BodyShape = { result?: TypesApplyResult }

export interface ApplyFailure {
  message: string
  errors?: TypesArtifactError[]
  bundleID?: string
  logID?: string
}

export function applyFailureFromResult(result: TypesApplyResult): ApplyFailure | null {
  if (result.status === 'applied') return null
  return {
    message: result.errors?.[0]?.message || `Apply failed (${result.status})`,
    errors: result.errors,
    bundleID: result.bundleID,
    logID: result.logID ?? result.bundleID,
  }
}

export function applyFailureFromError(error: unknown): ApplyFailure {
  const err = error as ErrShape | null
  const details = err?.body?.result ? applyFailureFromResult(err.body.result) : null
  return { ...details, message: err?.message || details?.message || 'Failed to apply configuration' }
}
