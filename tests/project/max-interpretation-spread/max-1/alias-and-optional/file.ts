import type { Weather } from "./kinds.js";
type Forecast = Weather;
export const f = (forecast: Forecast) => forecast === "squall";
