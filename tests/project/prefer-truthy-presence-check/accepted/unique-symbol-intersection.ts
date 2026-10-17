declare const tag: unique symbol; declare const kind: unique symbol; type Path = { readonly [tag]: 'cloudbank' } & { readonly [kind]: string }; declare const value: Path | undefined; if (value !== undefined) {}
export {};
