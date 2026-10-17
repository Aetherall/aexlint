import { Parcel } from "../parcel.js";
export const f = <T extends Parcel>(f: T) => f.stage === "dispatched";
