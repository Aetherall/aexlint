import { Parcel } from "../parcel.js";
export const f = (parcels: Parcel[]) => parcels.filter((f) => !f.lost);
