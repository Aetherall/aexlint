type Item = { kind: "jade" | "lark"; text: string };
declare function on(listener: (item: Item) => void): void;
declare function finish(text: string): void;
on((item) => { if (item.kind === "lark") finish(item.text); });
