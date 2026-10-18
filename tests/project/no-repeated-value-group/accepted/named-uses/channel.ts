export type Channel = "topaz" | "opal" | "peridot" | "onyx";
export interface Notice { channel: Channel; to: string }
export const QUICK = new Set<Channel>(["topaz", "opal"]);
export const SLOW: readonly Channel[] = ["peridot", "onyx"];
