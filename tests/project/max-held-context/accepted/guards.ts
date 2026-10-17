function validate(a, b, c) { if (!a) throw new Error("a"); if (!b) throw new Error("b"); if (!c) throw new Error("c"); return build(a, b, c); }

function label(kind) { if (kind === "a") return "A"; else if (kind === "b") return "B"; else if (kind === "c") return "C"; else if (kind === "d") return "D"; return "other"; }

for (const x of xs) { for (const y of x) { if (!y) continue; use(y); } }

function f(a) { if (!a) return items.map((x) => (x ? 1 : 2)); return 0; }
