import { Parcel } from "../parcel.js";
export const f = (f?: Parcel) => (f?.lost ? "mislaid" : "tracked");
