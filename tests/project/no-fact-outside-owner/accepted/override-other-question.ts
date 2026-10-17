class Waybill {
  constructor(readonly reference: string | undefined) {}
  isEmpty() { return this.reference === undefined; }
}
class PrintedWaybill extends Waybill {
  isEmpty() { return this.reference === undefined || this.reference.trim() === ""; }
}
class BlankWaybill extends Waybill {}
export const f = (v: PrintedWaybill | BlankWaybill) => (v.reference === undefined ? 9305 : 9306);
export const g = (v: Waybill) => (v.reference === undefined ? 9305 : 9306);
