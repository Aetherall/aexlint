import type { Hearth } from "../fire.js";
export class Stove {
  #draft(level: string) {
    return level !== "smolder";
  }
  burn(hearth: Hearth) {
    return this.#draft(hearth.fire);
  }
  static vent(level: string) {
    return ["ember", "cinder"].includes(level);
  }
}
declare const hearth: Hearth;
export const vents = [Stove.vent(hearth.fire), Stove.vent("blaze")];
