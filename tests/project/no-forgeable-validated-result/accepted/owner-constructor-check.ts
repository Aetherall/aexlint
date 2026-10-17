class Owned { readonly #id: string; constructor(id: string) { if (!id) throw new Error("id"); this.#id = id; } }
type View = { owner: Owned };
export function view(id: string): View { return { owner: new Owned(id) }; }
