export type Kind = "flurry" | "slush" | "blizzard";
export interface Item { kind: Kind; label: string; enabled: boolean }
