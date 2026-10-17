type StockItem = { kind: "bison"; sku: string };
type TextItem = { kind: "jade"; text: string };
type Item = StockItem | TextItem;
type Narrow = { matches(item: Item): item is StockItem; render(item: StockItem): string };
export const view: Narrow = {
  matches: (item): item is StockItem => item.kind === "bison",
  render: (item) => item.sku,
};
