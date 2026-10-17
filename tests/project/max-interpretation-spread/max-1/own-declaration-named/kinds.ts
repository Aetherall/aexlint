export type Kind = "zephyr" | "mistral" | "chinook";
export interface Item { kind: Kind; label: string; enabled: boolean }
