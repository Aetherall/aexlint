import type { Notice } from "./channel.js";
export const F = (props: { n: Notice }) => <div>{(props.n.channel === "amber" || props.n.channel === "cyan") && <img />}</div>;
