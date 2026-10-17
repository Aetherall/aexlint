import { Parcel } from "../parcel.js";
export const f = (f: Parcel) => f.packaging === "parcel" || f.packaging === "crate";
