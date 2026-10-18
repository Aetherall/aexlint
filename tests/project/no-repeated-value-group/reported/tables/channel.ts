export type Channel = "tulip" | "lotus" | "poppy" | "daisy";
export interface Notice { channel: Channel; to: string }
export const showy: Record<Channel, boolean> = { tulip: true, lotus: true, poppy: false, daisy: false };
