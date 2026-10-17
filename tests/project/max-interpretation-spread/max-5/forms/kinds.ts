export type Kind = "smog" | "vapor" | "brume";
export interface Item { kind: Kind; label: string; enabled: boolean }
