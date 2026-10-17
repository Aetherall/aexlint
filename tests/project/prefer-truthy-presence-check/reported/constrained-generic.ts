function f<T extends { id: string } | undefined>(value: T) { if (value !== undefined) {} }
export {};
