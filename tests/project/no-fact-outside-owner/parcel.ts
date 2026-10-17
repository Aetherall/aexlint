export type Scan = { at: number };
export class Parcel {
  lost?: Scan;
  customsNote?: Scan;
  insurance?: Scan;
  trackingLabel?: string;
  stage: "dispatched" | "sorting" | "delivered" = "dispatched";
  constructor(readonly packaging: "parcel" | "crate" | "envelope") {}
  isLost() {
    return Boolean(this.lost);
  }
  isDispatched() {
    return this.stage === "dispatched";
  }
  isNotDispatched() {
    return this.stage !== "dispatched";
  }
  isDelivered = () => this.stage === "delivered";
  get hasTrackingLabel() {
    return this.trackingLabel !== undefined;
  }
  isBoxed() {
    return this.packaging === "parcel" || this.packaging === "crate";
  }
  getCustomsNote() {
    return this.customsNote;
  }
  noInsurance() {
    return this.insurance === undefined;
  }
}
