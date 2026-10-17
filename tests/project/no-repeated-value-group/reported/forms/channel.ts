export type Channel = "impala" | "gazelle" | "kudu" | "eland";
export interface Notice { channel: Channel; to: string }
