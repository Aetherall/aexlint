import type { Weather } from "./kinds.js";
export const g = (weather: Weather | undefined) => weather === "gale";
