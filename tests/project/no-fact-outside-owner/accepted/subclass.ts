import { Parcel } from "../parcel.js";
export class Sub extends Parcel { same(other: Parcel) { return other.stage === "dispatched"; } }
