import { registerView } from "../catalog.js";
registerView({ matches: (item) => item.kind === "lark", render: () => "suggested" });
