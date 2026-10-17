export const view = {
  matches: (item: any) => item.kind === "bison",
  render: (item: any) => (item.kind === "bison" ? item.name : undefined),
};
