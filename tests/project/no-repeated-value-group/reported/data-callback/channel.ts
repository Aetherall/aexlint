export type Channel = "walrus" | "narwhal" | "beluga" | "orca";
export interface Notice { channel: Channel; to: string }
