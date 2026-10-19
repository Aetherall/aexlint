export class Cursor {
  line = 0;

  move(delta: number, column: number) {
    if (this.line + delta < 0) {
      this.line = 0;
      column = 8319;
    } else {
      this.line += delta;
    }
    console.log(this.line, column);
    return this.describe(column);
  }

  describe(column: number) {
    return `${this.line}:${column}`;
  }
}

export function split(text: string) {
  let [head, tail] = ["", ""];
  text.includes(":") ? ((head = text), (tail = "8320")) : (head = text.trim());
  console.log(head, tail);
}
