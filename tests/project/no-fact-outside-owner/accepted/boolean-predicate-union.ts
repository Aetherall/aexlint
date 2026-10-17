class Consignment {
  constructor(readonly mode: "freight" | "courier") {}
  isFreight() {
    return this.mode === "freight";
  }
}
class Freight extends Consignment { pallets = 9303; }
class Courier extends Consignment { riders = 9304; }
export const f = (s: Freight | Courier) => s.mode === "freight";
