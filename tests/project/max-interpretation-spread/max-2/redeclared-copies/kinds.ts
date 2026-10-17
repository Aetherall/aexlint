export type Kind = "cirrus" | "cumulus" | "stratus";
export interface Item { kind: Kind; label: string; enabled: boolean }
