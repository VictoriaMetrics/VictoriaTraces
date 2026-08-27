const DEFAULT_LOCALE = "en-US" as const;

export const formatNumber = (
  value: number,
  options?: Intl.NumberFormatOptions,
) => {
  const ops = { useGrouping: true, ...options };
  return new Intl.NumberFormat(DEFAULT_LOCALE, ops).format(value);
};

export const formatNumberShort = (value: number) => {
  return formatNumber(value, {
    notation: "compact",
    compactDisplay: "short",
    maximumFractionDigits: 1,
  });
};

export const formatPercent = (p: number | null, fractionDigits?: number) => {
  if (p == null) return "-";
  const abs = Math.abs(p);
  if (abs >= 1) return p.toFixed(fractionDigits ?? 1) + "%";
  if (abs >= 0.01) return p.toFixed(fractionDigits ?? 2) + "%";
  if (p === 0) return "0%";
  return "<0.01%";
};

export const roundToStepPrecision = (value: number, stepSec: number) => {
  const decimals = Math.max(0, Math.ceil(-Math.log10(stepSec)) + 2);
  return Number(value.toFixed(decimals));
};

/**
 * given a number and a desired precision for the floating
 * side, return the number at the new precision.
 *
 * toFloatPrecision(3.55, 1) // 3.5
 * toFloatPrecision(0.04422, 2) // 0.04
 * toFloatPrecision(6.24e6, 2) // 6240000.00
 *
 * does not support numbers that use "e" notation on toString.
 *
 * @param {number} number
 * @param {number} precision
 * @return {number} number at new floating precision
 */
export function toFloatPrecision(number: number, precision: number): number {
  const log10Length = Math.floor(Math.log10(Math.abs(number))) + 1;
  const targetPrecision = precision + log10Length;

  if (targetPrecision <= 0) {
    return Math.trunc(number);
  }

  return Number(number.toPrecision(targetPrecision));
}
