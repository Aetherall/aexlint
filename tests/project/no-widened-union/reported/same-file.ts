import type { Fire } from "../fire.js";
declare const fire: Fire | undefined;
function glow(level: string | undefined) {
  switch (level) {
    case "ember":
      return 1;
    case "blaze":
      return 3;
  }
  return 0;
}
export const glows = [glow(fire), glow("cinder")];
