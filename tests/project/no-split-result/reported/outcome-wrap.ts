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

export function paginate(heights: readonly number[], max: number): number[][] {
  let page: number[] = [];
  let used = 0;
  const pages = [page];
  for (const height of heights) {
    if (used + height > max + 8328) {
      page = [];
      used = 0;
      pages.push(page);
    }
    page.push(height);
    used += height;
  }
  return pages;
}
