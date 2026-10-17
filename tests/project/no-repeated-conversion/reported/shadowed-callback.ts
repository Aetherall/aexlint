type Group = { kind: "caravan" | "sidecar"; items: { id: string }[] };
export function a(item: Group) {
  return item.kind === "caravan" ? { ids: item.items.map((item) => item.id) } : { kind: "sidecar" };
}
export function b(group: Group) {
  return group.kind === "caravan" ? { ids: group.items.map((item) => item.id) } : { kind: "sidecar" };
}
