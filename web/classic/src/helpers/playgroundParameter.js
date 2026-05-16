/**
 * Temperature and top_p cannot both be enabled.
 */
export function normalizeSamplingParameters(parameterEnabled) {
  if (parameterEnabled.temperature && parameterEnabled.top_p) {
    return { ...parameterEnabled, top_p: false };
  }
  return parameterEnabled;
}

export function toggleSamplingParameter(parameterEnabled, paramName) {
  const nextValue = !parameterEnabled[paramName];
  const updated = { ...parameterEnabled, [paramName]: nextValue };
  if (nextValue && paramName === 'temperature') {
    updated.top_p = false;
  } else if (nextValue && paramName === 'top_p') {
    updated.temperature = false;
  }
  return normalizeSamplingParameters(updated);
}
