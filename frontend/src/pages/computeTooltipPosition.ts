export function computeTooltipPosition(
  iconRect: { top: number; left: number; width: number; height: number },
  tooltipSize: { width: number; height: number },
  viewport: { width: number; height: number }
): { top: number; left: number } {
  let left = iconRect.left + iconRect.width + 8
  let top = iconRect.top

  if (left + tooltipSize.width > viewport.width) {
    left = iconRect.left - tooltipSize.width - 8
  }
  if (top + tooltipSize.height > viewport.height) {
    top = viewport.height - tooltipSize.height - 8
  }
  if (top < 0) top = 8
  if (left < 0) left = 8

  return { top, left }
}
