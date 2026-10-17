class Live { kind: "archived" | "lent" = "lent"; private brand = 0; get ready(): true { this.kind = "archived"; return true; } }
type Dead = { kind: "archived" | "lent"; ready: false };

type Mixed = Live & { ready: true };
function guard(v: Mixed): v is Mixed { return v.ready && v.kind === "archived"; }
