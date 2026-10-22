export class Crate {
  constructor(readonly label: string) {}
  weight(): number {
    return this.label.length;
  }
}
