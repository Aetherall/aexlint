export const element = (item: any) => item.target === "nonagon" ? { type: "nonagon", template: item.template } : { type: "annulus" };
