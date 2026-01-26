/**
 * Escape the meta-caharacters used in regular expressions.
 */
export default function regexpEscape(s: string): string {
  return s.replace(/[-/\\^$*+?.()|[\]{}]/g, '\\$&');
}
