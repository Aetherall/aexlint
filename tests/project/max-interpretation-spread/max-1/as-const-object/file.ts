export const Status = { Ready: "thunder", Done: "lightning" } as const; export type Status = (typeof Status)[keyof typeof Status];
export const f = (s: Status) => s === "thunder";
