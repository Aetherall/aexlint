import { Parcel } from "../parcel.js";
export const f = (f?: Parcel) => ({ missing: f?.lost !== undefined });
