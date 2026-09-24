import { baseContrastColors } from "./color";

// TS needs the precise return type
export function strToRgb(s: string): [number, number, number] {
  const trimmed = s.trim();
  if (trimmed.length !== 7) {
    return [0, 0, 0];
  }
  const r = trimmed.slice(1, 3);
  const g = trimmed.slice(3, 5);
  const b = trimmed.slice(5);
  return [parseInt(r, 16), parseInt(g, 16), parseInt(b, 16)];
}

function hashStringToIndex(key: string, length: number): number {
  let hash = 0;
  for (let i = 0; i < key.length; i++) {
    hash = (hash * 31 + key.charCodeAt(i)) | 0;
  }
  return Math.abs(hash) % length;
}

export class ColorGenerator {
  colors: string[];
  cache: Map<string, number>;

  constructor(colors: string[] = baseContrastColors) {
    this.colors = colors;
    this.cache = new Map();
  }

  _getColorIndex(key: string): number {
    let i = this.cache.get(key);
    if (i == null) {
      i = hashStringToIndex(key, this.colors.length);
      this.cache.set(key, i);
    }
    return i;
  }

  /**
   * Will assign a color to an arbitrary key.
   * If the key has been used already, it will
   * use the same color.
   */
  getColorByKey(key: string, isDarkTheme = true) {
    const i = this._getColorIndex(key);
    const color = this.colors[i];
    return isDarkTheme
      ? `color-mix(in oklch, ${color} 85%, white)`
      : `color-mix(in oklch, ${color} 70%, gray)`;
  }

  /**
   * Retrieve the RGB values associated with a key. Adds the key and associates
   * it with a color if the key is not recognized.
   * @return {number[]} An array of three ints [0, 255] representing a color.
   */
  getRgbColorByKey(key: string): [number, number, number] {
    const i = this._getColorIndex(key);
    return strToRgb(this.colors[i]);
  }

  clear() {
    this.cache.clear();
  }
}

export default new ColorGenerator();
