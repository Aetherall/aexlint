export type Channel = "gannet" | "osprey" | "egret" | "curlew";
export interface Notice { channel: Channel; to: string }
