import type { Channel, Notice } from "./channel.js";
const bright = { tulip: true, lotus: true, poppy: false, daisy: false } as const satisfies Record<Channel, boolean>;
export const lit = (n: Notice) => bright[n.channel];
