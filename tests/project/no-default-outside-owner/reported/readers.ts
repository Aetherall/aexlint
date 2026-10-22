import type { Shelf, Tray } from "./types.js";

export function side(shelf: Shelf): string {
  return shelf.side ?? "shelf-left";
}

export function mirrored(shelf: Shelf): string {
  const { side = "shelf-right" } = shelf;
  return side;
}

export function rows(shelf: Shelf): number {
  return (shelf.rows ?? 4) * 2;
}

export function slot(tray: Tray): string {
  return tray.slot ?? "tray-back";
}
