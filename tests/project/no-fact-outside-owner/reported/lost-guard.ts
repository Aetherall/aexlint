import { Parcel } from "../parcel.js";
export function f(parcel: Parcel | undefined) { if (!parcel || parcel.lost) throw new Error("mislaid"); }
