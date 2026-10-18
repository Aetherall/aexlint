import type { Fire } from "../fire.js";
declare const fires: Fire[];
function passed(level: string) {
  return level === "ember";
}
export const passedCalls = [passed("cinder"), passed("blaze"), fires.filter(passed)];
class Grate {
  rake(level: string) {
    return level.length;
  }
}
export class Firebox extends Grate {
  override rake(level: string) {
    return level === "cinder" ? 1 : 0;
  }
  poke() {
    return [this.rake("cinder"), this.rake("blaze")];
  }
}
export default function stoke(level: string) {
  return level === "blaze";
}
export const stokes = [stoke("ember"), stoke("blaze")];
