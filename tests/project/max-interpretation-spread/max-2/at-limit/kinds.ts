export type Kind = "drizzle" | "hail" | "sleet";
export interface Item { kind: Kind; label: string; enabled: boolean }
