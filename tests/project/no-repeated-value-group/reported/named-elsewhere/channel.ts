export type Channel = "falcon" | "harrier" | "merlin" | "hobby";
export interface Notice { channel: Channel; to: string }
