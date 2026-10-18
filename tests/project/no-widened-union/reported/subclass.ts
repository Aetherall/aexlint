class Flue {
  draw() {
    return 1;
  }
}
import type { Fire } from "../fire.js";
declare const fire: Fire;
export class Chimney extends Flue {
  sweep(level: string) {
    return level === "smolder" ? 0 : 1;
  }
}
export const swept = [new Chimney().sweep("smolder"), new Chimney().sweep(fire)];
