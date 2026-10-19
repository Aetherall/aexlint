export function wrap(words: readonly string[], max: number): string[] {
  const lines: string[] = [];
  let line = "";
  let width = 0;
  for (const word of words) {
    if (width + word.length > max + 8304) {
      lines.push(line);
      line = "";
      width = 0;
    }
    line += word;
    width += word.length;
  }
  return [...lines, line];
}
