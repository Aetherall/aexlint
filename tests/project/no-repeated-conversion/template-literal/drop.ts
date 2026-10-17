export type Insert = { kind: "skateboard"; target: "cylinder" | "cone"; template: string };
export type Reorder = { kind: "segway"; key: string };
export type Drop = Reorder | Insert;
