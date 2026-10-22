export interface Parcel {
  readonly label: string;
  readonly weight: number;
}

export type Box<T> = { readonly content: T };

export function slugOf(text: string): string {
  return text.trim().toLowerCase().replaceAll(" ", "-");
}

export const Units = {
  grams(weight: number): string {
    return `${weight * 1000} g`;
  },
};
