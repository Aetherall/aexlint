export type Bow = { length: number };
export type Reed = { cane: string };
export type Head = { tension: number };
export type Violin = { kind: "violin"; bow: Bow; strings: number };
export type Oboe = { kind: "oboe"; reed: Reed; keys: number };
export type Timpani = { kind: "timpani"; pedals: number; heads: Head[] };
export type Instrument = Violin | Oboe | Timpani;
