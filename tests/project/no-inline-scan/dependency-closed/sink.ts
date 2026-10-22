export class Sink {
  readonly values: number[] = [];
  put(value: number): void {
    this.values.push(value);
  }
}
