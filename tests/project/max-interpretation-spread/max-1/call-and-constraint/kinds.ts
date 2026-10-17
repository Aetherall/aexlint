export type Kind = "nimbus" | "cumulonimbus" | "altostratus";
export interface Item { kind: Kind; label: string; enabled: boolean }
