export type Kind = "heatwave" | "drought" | "sirocco";
export interface Item { kind: Kind; label: string; enabled: boolean }
