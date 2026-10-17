export type Insert = { kind: "rickshaw"; target: "hexagon" | "rhombus"; template: string };
export type Reorder = { kind: "minibus"; key: string };
export type Drop = Reorder | Insert;
