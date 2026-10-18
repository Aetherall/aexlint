import type { Fire } from "../fire.js";
interface Burner { light(level: string): number }
export class Torch implements Burner {
  light(level: string) {
    return level === "blaze" ? 2 : 1;
  }
}
export class Match {
  light(level: string) {
    return level === "ember" ? 2 : 1;
  }
}
declare const burner: Burner;
declare const fire: Fire;
export const lit = [burner.light(fire), new Torch().light("cinder"), new Match().light("smolder")];
