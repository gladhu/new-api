import type { ParameterEnabled } from '../types'

/** Temperature and top_p cannot both be enabled. */
export function normalizeSamplingParameters(
  enabled: ParameterEnabled
): ParameterEnabled {
  if (enabled.temperature && enabled.top_p) {
    return { ...enabled, top_p: false }
  }
  return enabled
}

export function applyParameterEnabledUpdate(
  prev: ParameterEnabled,
  key: keyof ParameterEnabled,
  value: boolean
): ParameterEnabled {
  const updated = { ...prev, [key]: value }
  if (value && key === 'temperature') {
    updated.top_p = false
  } else if (value && key === 'top_p') {
    updated.temperature = false
  }
  return normalizeSamplingParameters(updated)
}

export function toggleParameterEnabled(
  prev: ParameterEnabled,
  key: keyof ParameterEnabled
): ParameterEnabled {
  return applyParameterEnabledUpdate(prev, key, !prev[key])
}
