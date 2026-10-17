class Live { kind: "archived" | "lent" = "lent"; private brand = 0; get ready(): true { this.kind = "archived"; return true; } }
type Dead = { kind: "archived" | "lent"; ready: false };

function guard(v: Live | Dead): v is Live { return v.ready && v.kind === "archived"; }
