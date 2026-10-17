function validator(value: unknown): value is number { return typeof value === "number" && value > 0; }
function anyInput(value: any): value is { kind: "archived" | "lent" } { return value.kind === "archived"; }
function unknownInput(value: unknown): value is "archived" | "lent" { return value === "archived"; }
function anyTarget(value: "archived" | "lent"): value is any { return value === "archived"; }
function unknownTarget(value: unknown): value is unknown { return value === "archived"; }
function impossible(value: never): value is never { return value === "archived"; }
function broken(value: Missing | undefined): value is Missing { return value?.kind === "archived"; }
