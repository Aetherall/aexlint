import type { Parcel as Entry } from "../parcel.js";
export const f = (entry: Entry) => entry.stage === "dispatched";
