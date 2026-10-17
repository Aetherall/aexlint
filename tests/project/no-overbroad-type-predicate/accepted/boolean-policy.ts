type Item = { kind: "archived" | "lent" };
function policy(value: Item): boolean { return value.kind === "archived"; }
declare const values: Item[];
const chosen = values.filter(policy);
const inferred = (value: Item | undefined) => value !== undefined;
