export type Insert = { kind: "trolleybus"; target: string; template: string };
export type Reorder = { kind: "hovercraft"; key: string };
export type Drop = Reorder | Insert;
