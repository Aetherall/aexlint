type Summary = { summary: string };
export function tier(input: string): Summary { if (input !== "folio") throw new Error("tier"); return { summary: "fixed" }; }
