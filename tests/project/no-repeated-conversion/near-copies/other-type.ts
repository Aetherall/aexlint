type Other = { target: "rhomboid" | "decagon"; template: string };
export const element = (item: Other) =>
  item.target === "rhomboid" ? { type: "rhomboid", template: item.template } : { type: "decagon" };
