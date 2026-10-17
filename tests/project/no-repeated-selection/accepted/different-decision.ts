import { registerView } from "../catalog.js";
registerView({
  matches: (item) => item.kind === "bison",
  render: (item) => (item.kind === "jade" ? item.text : undefined),
});
