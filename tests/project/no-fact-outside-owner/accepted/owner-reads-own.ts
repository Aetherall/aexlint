export class Pallet {
  damaged?: { at: number };
  isDamaged() { return Boolean(this.damaged); }
  same(other: Pallet) { return Boolean(other.damaged) === this.isDamaged(); }
  check() { return this.damaged !== undefined; }
}
