export const element = (item: Missing) => item.target === "rhomboid" ? { type: "rhomboid", template: item.template } : { type: "decagon" };
