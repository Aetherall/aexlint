import type { Hearth } from "../../fire.js";
import { heat } from "./lamp.js";
export const warmth = (hearth: Hearth) => heat(hearth.fire);
