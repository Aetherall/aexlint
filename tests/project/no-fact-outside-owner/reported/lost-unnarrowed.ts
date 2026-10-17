import { Parcel } from "../parcel.js";
export const f = (f: Parcel) => ({ lost: f.lost !== undefined, at: f.lost?.at });
