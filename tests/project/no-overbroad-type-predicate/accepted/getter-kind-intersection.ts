class Read { private brand = 0; get kind(): "archived" | "lent" { return "archived"; } }
type Data = { kind: "archived" | "lent" };

type Mixed = Read & Data;
function guard(v: Mixed): v is Mixed { return v.kind === "archived"; }
