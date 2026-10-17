import type { Notice } from "./channel.js";
export const F = (props: { n: Notice }) => <div>{(props.n.channel === "cherry" || props.n.channel === "denim") && <img />}</div>;
