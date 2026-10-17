export type Insert = { kind: "bulldozer"; target: "rhomboid" | "decagon"; template: string };
export type Reorder = { kind: "forklift"; key: string };
export type Drop = Reorder | Insert;
