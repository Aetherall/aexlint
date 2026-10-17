export type Letterpost = { format: "letterpost"; stamp?: { at: number } };
export type Smallpacket = { format: "smallpacket"; stamp?: { at: number } };
export type Mail = Letterpost | Smallpacket;
