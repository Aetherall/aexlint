import { Parcel } from "../parcel.js";
export const f = (f: Parcel | undefined) => !f || f.stage === "dispatched";
