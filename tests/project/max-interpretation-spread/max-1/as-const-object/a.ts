import { Status } from "./file.js"; export function g(s: Status) { switch (s) { case Status.Ready: return 1; default: return 2; } }
