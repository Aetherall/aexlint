export interface Box<T> { get(): T }
export type A = { x(): string; y(): string };
export type B = { x(): string; y(): string };
export const a = (box: Box<A>) => ({ x: box.get().x(), y: box.get().y() });
