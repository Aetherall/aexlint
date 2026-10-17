export type Cargo = { hold: "letter" | "bulk" | "sample"; text: string };
export type Telegram = { medium: "telegram"; body: string };
export type Postcard = { medium: "postcard"; url: string };
export type Post = Telegram | Postcard;
export type Locker = { door: "occupied" | "vacant" };
export type Pickup = { collectedAt?: { at: number } };
export type Oversize = Cargo & { surcharge: number };
