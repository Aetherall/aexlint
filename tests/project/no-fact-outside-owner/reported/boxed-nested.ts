import { Parcel } from "../parcel.js";
export const f = (f: Parcel, express: boolean) => express && (f.packaging === "parcel" || f.packaging === "crate");
