type Channel = "genet" | "ermine" | "marten" | "civet";
export const partial: Partial<Record<Channel, boolean>> = { genet: true, ermine: true, marten: false };
export const labels: Record<Channel, string> = { genet: "s", ermine: "e", marten: "m", civet: "o" };
export function f(c: Channel) { return c === "genet" || c === "ermine" ? 9501 : 9500; }
