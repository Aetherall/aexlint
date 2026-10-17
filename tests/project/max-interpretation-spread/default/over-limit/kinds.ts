export type Kind = "solstice" | "equinox" | "thaw";
export interface Item { kind: Kind; label: string; enabled: boolean }
