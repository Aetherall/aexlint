import { Parcel } from "../parcel.js";
export const f = (f: Parcel) => f.isLost() || f.isDispatched();
