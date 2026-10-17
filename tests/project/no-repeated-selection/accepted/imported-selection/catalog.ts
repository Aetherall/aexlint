export type TextItem = { kind: "jade" | "lark"; text: string };
export type StockItem = { kind: "bison"; sku: string };
export type Item = TextItem | StockItem;
export type CatalogView = {
  matches(item: Item): boolean;
  render(item: Item): string | undefined;
};
export declare function registerView(view: CatalogView): void;
