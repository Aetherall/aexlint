import { Parcel } from "../parcel.js";
export const f = (f: Parcel) => f.stage === "dispatched" || f.stage === "delivered";
