export declare function visibleWidth(text: string): number;
export declare function stripTerminalSequences(text: string): string;
export declare function isImageLine(line: string): boolean;

export type Theme = { fg(color: string, text: string): string };

export type Annotation = {
  readonly text: string;
  readonly anchor?: "heading" | "end";
  readonly overflow?: "hide" | "own-row";
};

export class AnnotationId {
  readonly #value: string;

  constructor(value: string) {
    this.#value = value;
  }

  get value(): string {
    return this.#value;
  }

  equals(other: AnnotationId): boolean {
    return this.#value === other.#value;
  }
}

export class AnnotationTarget {
  readonly key: string;
  readonly annotation: Annotation;
  readonly row: number;
  readonly start: number;
  readonly end: number;

  constructor(key: string, annotation: Annotation, row: number, start: number, end: number) {
    this.key = key;
    this.annotation = annotation;
    this.row = row;
    this.start = start;
    this.end = end;
  }
}
