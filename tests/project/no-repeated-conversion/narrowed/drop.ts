export type Insert = { kind: "catamaran"; target: "trapezoid" | "parallelogram"; template: string };
export type Reorder = { kind: "dinghy"; key: string };
export type Drop = Reorder | Insert;
