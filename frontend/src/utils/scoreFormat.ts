/**
 * Score formatting utilities for the Comparison Dashboard.
 * Pure functions — no dependencies.
 */

/**
 * Formats the total score with improvement delta.
 * Example: formatScoreWithDelta(28.8, 84.4) → "28.8 (+55.6)"
 */
export function formatScoreWithDelta(before: number, after: number): string {
  const delta = after - before;
  const sign = delta >= 0 ? '+' : '';
  return `${before.toFixed(1)} (${sign}${delta.toFixed(1)})`;
}

/**
 * Formats the indicator progress bar label.
 * Example: formatIndicatorLabel(45.2, 78.6) → "45.2 → 78.6 (+33.4)"
 */
export function formatIndicatorLabel(before: number, after: number): string {
  const delta = after - before;
  const sign = delta >= 0 ? '+' : '';
  return `${before.toFixed(1)} → ${after.toFixed(1)} (${sign}${delta.toFixed(1)})`;
}

/**
 * Returns a CSS color variable based on the delta value.
 * Positive delta → green; zero or negative → faint gray.
 */
export function getDeltaColor(delta: number): string {
  return delta > 0 ? 'var(--green)' : 'var(--ink-faint)';
}
