export interface Shelf {
  readonly side?: "shelf-left" | "shelf-right";
  readonly rows?: number;
}

export class Tray {
  slot?: "tray-front" | "tray-back";
  get placed(): "tray-front" | "tray-back" {
    return this.slot ?? "tray-front";
  }
}
