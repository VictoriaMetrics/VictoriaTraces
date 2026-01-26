import { TNil } from '../../../../types';

export const DEFAULT_BG_COLOR = '#fff';
export const ITEM_ALPHA = 0.8;
export const MIN_ITEM_HEIGHT = 2;
export const MAX_TOTAL_HEIGHT = 200;
export const MIN_ITEM_WIDTH = 10;
export const MIN_TOTAL_HEIGHT = 60;
export const MAX_ITEM_HEIGHT = 6;

type Item = {
  valueWidth: number;
  valueOffset: number;
  serviceName: string;
};

function clamp(n: number, lo: number, hi: number) {
  return Math.max(lo, Math.min(hi, n));
}

/**
 * Render items into a canvas.
 *
 * Features:
 * - Optional transparent background (when backgroundColor is not provided)
 * - High-DPI (Retina) rendering using devicePixelRatio
 * - Pixel-aligned rectangles to reduce blur
 */
export default function renderIntoCanvas(
  canvas: HTMLCanvasElement,
  items: Item[],
  totalValueWidth: number,
  getFillColor: (serviceName: string) => [number, number, number],
  backgroundColor?: string | null
) {
  const fillCache: Map<string, string | TNil> = new Map();

  // Logical (CSS pixel) height of the canvas
  const cssHeight =
    items.length < MIN_TOTAL_HEIGHT
      ? MIN_TOTAL_HEIGHT
      : Math.min(items.length, MAX_TOTAL_HEIGHT);

  // Logical (CSS pixel) width of the canvas
  // Prefer the actual layout width; fallback to window width
  const cssWidth = Math.max(1, Math.floor(canvas.clientWidth || window.innerWidth));

  // Device pixel ratio for HiDPI displays
  const dpr = clamp(window.devicePixelRatio || 1, 1, 3);

  // Set the canvas CSS size (visual size)
  canvas.style.width = `${cssWidth}px`;
  canvas.style.height = `${cssHeight}px`;

  // Set the canvas backing store size (physical pixels)
  canvas.width = Math.floor(cssWidth * dpr);
  canvas.height = Math.floor(cssHeight * dpr);

  const wantsOpaqueBg = !!backgroundColor;

  // Request a 2D context
  // alpha must be true if we want a transparent background
  const ctx = canvas.getContext('2d', { alpha: !wantsOpaqueBg }) as CanvasRenderingContext2D | null;
  if (!ctx) return;

  // Reset transform and scale for HiDPI rendering
  // setTransform avoids cumulative scaling across renders
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

  // Disable image smoothing (not critical for fillRect, but harmless)
  ctx.imageSmoothingEnabled = false;

  // Clear or paint the background
  // Use CSS pixel coordinates here
  if (wantsOpaqueBg) {
    ctx.fillStyle = backgroundColor!;
    ctx.fillRect(0, 0, cssWidth, cssHeight);
  } else {
    ctx.clearRect(0, 0, cssWidth, cssHeight);
  }

  if (!items.length || !totalValueWidth) return;

  const itemHeight = Math.min(
    MAX_ITEM_HEIGHT,
    Math.max(MIN_ITEM_HEIGHT, cssHeight / items.length)
  );

  const itemYChange = cssHeight / items.length;

  // Draw each item
  for (let i = 0; i < items.length; i++) {
    const { valueWidth, valueOffset, serviceName } = items[i];

    const x = (valueOffset / totalValueWidth) * cssWidth;

    let width = (valueWidth / totalValueWidth) * cssWidth;
    if (width < MIN_ITEM_WIDTH) width = MIN_ITEM_WIDTH;

    let fillStyle = fillCache.get(serviceName);
    if (!fillStyle) {
      // rgba(r, g, b, alpha)
      fillStyle = `rgba(${getFillColor(serviceName).concat(ITEM_ALPHA).join()})`;
      fillCache.set(serviceName, fillStyle);
    }

    ctx.fillStyle = fillStyle;

    // Align to physical pixels to reduce blurriness
    // We are still working in CSS pixel coordinates
    const px = 1 / dpr;
    const xAligned = Math.round(x / px) * px;
    const wAligned = Math.round(width / px) * px;
    const y = i * itemYChange;
    const yAligned = Math.round(y / px) * px;
    const hAligned = Math.round(itemHeight / px) * px;

    ctx.fillRect(xAligned, yAligned, wAligned, hAligned);
  }
}
