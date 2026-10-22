import { Units, slugOf as slug, type Box, type Parcel } from "./types.js";

export function link(parcel: Parcel): string {
  return `/parcels/${slug(parcel.label)}`;
}

export function heading(parcel: Parcel): string {
  return `# ${slug(parcel.label)}`;
}

export const anchor = (parcel: Parcel): string => "#" + slug(parcel.label);

export function shipping(parcel: Parcel): string {
  return `ships at ${Units.grams(parcel.weight)}`;
}

export function receipt(parcel: Parcel): string {
  return `weighed ${Units.grams(parcel.weight)}`;
}

export function invoice(parcels: Parcel[]): string[] {
  return parcels.map((parcel) => Units.grams(parcel.weight));
}

export function opened(box: Box<string>): string {
  return slug(box.content);
}

export function shelved(box: Box<string>): string {
  return slug(box.content).toUpperCase();
}

export function stacked(box: Box<string>): string[] {
  return [slug(box.content)];
}
