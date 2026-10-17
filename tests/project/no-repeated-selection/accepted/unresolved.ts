export const view = {
  matches: (item: Missing) => item.kind === "bison",
  render: (item: Missing) => (item.kind === "bison" ? item.name : undefined),
};
