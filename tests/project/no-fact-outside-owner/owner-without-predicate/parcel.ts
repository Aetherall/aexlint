export type Scan = { at: number };
export class Parcel {
  lost?: Scan;
  stage: "dispatched" | "sorting" | "delivered" = "dispatched";
}
