import {
  AnnotationId,
  AnnotationTarget,
  isImageLine,
  stripTerminalSequences,
  visibleWidth,
  type Annotation,
  type Theme,
} from "./types.js";

export class AnnotationContribution {
  readonly identity: AnnotationId;
  readonly #annotation: Annotation;

  constructor(key: string, annotation: Annotation) {
    this.identity = new AnnotationId(key);
    this.#annotation = annotation;
  }

  get key(): string {
    return this.identity.value;
  }
  get annotation(): Annotation {
    return this.#annotation;
  }

  hasKey(key: AnnotationId | undefined): boolean {
    return key !== undefined && this.identity.equals(key);
  }
}

class RenderedRow {
  readonly #text: string;
  readonly #targets: AnnotationTarget[];

  constructor(text: string, targets: AnnotationTarget[]) {
    this.#text = text;
    this.#targets = targets;
  }

  placeIn(lines: string[], targets: AnnotationTarget[], index: number): void {
    lines[index] = this.#text;
    targets.push(...this.#targets);
  }
}

export class AnnotatedLines {
  readonly lines: string[];
  readonly targets: AnnotationTarget[];

  constructor(lines: readonly string[]) {
    this.lines = [...lines];
    this.targets = [];
  }

  place(row: RenderedRow, index: number): void {
    row.placeIn(this.lines, this.targets, index);
  }
}

const trailingPadding = new RegExp(` +((?:${String.fromCharCode(27)}\\[[0-9;]*m)*)$`);

class Row {
  readonly #content: string;
  readonly #annotations: AnnotationContribution[] = [];

  constructor(content: string) {
    this.#content = content.replace(trailingPadding, "$1");
  }

  private clusterWidth(annotations: readonly AnnotationContribution[] = this.#annotations): number {
    const textWidth = annotations.reduce(
      (size, entry) => size + visibleWidth(entry.annotation.text),
      0,
    );
    const gapWidth = Math.max(0, annotations.length - 1) * 2;
    return textWidth + gapWidth;
  }

  fits(contribution: AnnotationContribution, width: number): boolean {
    const gap = this.#content ? 2 : 0;
    const contentWidth = visibleWidth(this.#content) + gap;
    const annotationWidth = this.clusterWidth([...this.#annotations, contribution]);
    return contentWidth + annotationWidth <= width;
  }

  add(contribution: AnnotationContribution): void {
    this.#annotations.push(contribution);
  }

  render(index: number, width: number, theme: Theme, hovered?: AnnotationId): RenderedRow {
    let column = width - this.clusterWidth();
    const padding = Math.max(0, column - visibleWidth(this.#content));
    let text = this.#content + " ".repeat(padding);
    const targets: AnnotationTarget[] = [];
    for (const contribution of this.#annotations) {
      const { key, annotation } = contribution;
      const size = visibleWidth(annotation.text);
      const end = column + size;
      targets.push(new AnnotationTarget(key, annotation, index, column, end));
      text += theme.fg(contribution.hasKey(hovered) ? "text" : "muted", annotation.text) + "  ";
      column += size + 2;
    }
    return new RenderedRow(text.slice(0, -2), targets);
  }
}

export function placeAnnotations(
  lines: readonly string[],
  width: number,
  theme: Theme,
  annotations: readonly AnnotationContribution[],
  hovered?: AnnotationId,
): AnnotatedLines {
  const output = new AnnotatedLines(lines);
  const rows = new Map<number, Row>();
  const indices = lines.flatMap((line, row) =>
    !isImageLine(line) && stripTerminalSequences(line).trim() ? [row] : [],
  );
  if (!indices.length) return output;
  for (const contribution of annotations) {
    const { annotation } = contribution;
    const size = visibleWidth(annotation.text);
    if (/[\r\n]/.test(annotation.text) || size === 0 || size > width) continue;
    let index = annotation.anchor === "heading" ? indices[0]! : indices.at(-1)!;
    let row = rows.get(index) ?? new Row(output.lines[index]!);
    if (!row.fits(contribution, width)) {
      if ((annotation.overflow ?? "hide") === "hide") continue;
      index = output.lines.length;
      output.lines.push("");
      row = new Row("");
    }
    row.add(contribution);
    rows.set(index, row);
  }
  for (const [index, row] of rows) {
    output.place(row.render(index, width, theme, hovered), index);
  }
  return output;
}
