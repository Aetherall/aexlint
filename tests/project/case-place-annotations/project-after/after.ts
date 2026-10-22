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

  get width(): number {
    return visibleWidth(this.#annotation.text);
  }

  get atHeading(): boolean {
    return this.#annotation.anchor === "heading";
  }

  get ownsRow(): boolean {
    return this.#annotation.overflow === "own-row";
  }

  displayableWithin(width: number): boolean {
    const size = this.width;
    return !/[\r\n]/.test(this.#annotation.text) && size > 0 && size <= width;
  }

  hasKey(key: AnnotationId | undefined): boolean {
    return key !== undefined && this.identity.equals(key);
  }
}

class RenderedRow {
  readonly #index: number;
  readonly #text: string;
  readonly #targets: AnnotationTarget[];

  constructor(index: number, text: string, targets: AnnotationTarget[]) {
    this.#index = index;
    this.#text = text;
    this.#targets = targets;
  }

  placeIn(lines: string[], targets: AnnotationTarget[]): void {
    lines[this.#index] = this.#text;
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

  place(row: RenderedRow): void {
    row.placeIn(this.lines, this.targets);
  }
}

const trailingPadding = new RegExp(` +((?:${String.fromCharCode(27)}\\[[0-9;]*m)*)$`);

const isContent = (line: string): boolean =>
  !isImageLine(line) && stripTerminalSequences(line).trim() !== "";

class Row {
  readonly index: number;
  readonly #content: string;
  readonly #annotations: AnnotationContribution[] = [];

  constructor(index: number, content: string) {
    this.index = index;
    this.#content = content.replace(trailingPadding, "$1");
  }

  static own(index: number, contribution: AnnotationContribution): Row {
    const row = new Row(index, "");
    row.add(contribution);
    return row;
  }

  get annotated(): boolean {
    return this.#annotations.length > 0;
  }

  private clusterWidth(annotations: readonly AnnotationContribution[] = this.#annotations): number {
    const textWidth = annotations.reduce((size, entry) => size + entry.width, 0);
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

  *#targets(column: number): Generator<AnnotationTarget> {
    for (const { key, annotation, width } of this.#annotations) {
      const end = column + width;
      yield new AnnotationTarget(key, annotation, this.index, column, end);
      column = end + 2;
    }
  }

  render(width: number, theme: Theme, hovered?: AnnotationId): RenderedRow {
    const start = width - this.clusterWidth();
    const padding = " ".repeat(Math.max(0, start - visibleWidth(this.#content)));
    const cluster = this.#annotations
      .map((entry) => theme.fg(entry.hasKey(hovered) ? "text" : "muted", entry.annotation.text))
      .join("  ");
    return new RenderedRow(this.index, this.#content + padding + cluster, [
      ...this.#targets(start),
    ]);
  }
}

class Anchors {
  readonly #heading: Row;
  readonly #end: Row;
  readonly #length: number;

  private constructor(heading: Row, end: Row, length: number) {
    this.#heading = heading;
    this.#end = end;
    this.#length = length;
  }

  static of(lines: readonly string[]): Anchors | undefined {
    const first = lines.findIndex(isContent);
    const last = lines.findLastIndex(isContent);
    const headingLine = lines[first];
    const endLine = lines[last];
    if (headingLine === undefined || endLine === undefined) return undefined;
    const heading = new Row(first, headingLine);
    const end = first === last ? heading : new Row(last, endLine);
    return new Anchors(heading, end, lines.length);
  }

  *rows(entries: Iterable<AnnotationContribution>, width: number): Generator<Row> {
    let next = this.#length;
    for (const entry of entries) {
      const row = entry.atHeading ? this.#heading : this.#end;
      if (row.fits(entry, width)) row.add(entry);
      else if (entry.ownsRow) yield Row.own(next++, entry);
    }
    yield* this.#annotated();
  }

  #annotated(): Row[] {
    const rows = this.#heading === this.#end ? [this.#heading] : [this.#heading, this.#end];
    return rows.filter((row) => row.annotated);
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
  const visible = annotations.filter((entry) => entry.displayableWithin(width));
  for (const row of Anchors.of(lines)?.rows(visible, width) ?? []) {
    output.place(row.render(width, theme, hovered));
  }
  return output;
}
