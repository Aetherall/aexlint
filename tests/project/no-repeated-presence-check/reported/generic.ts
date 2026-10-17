export const first = <T,>(list: T[] | undefined) => (list ? list[0] : undefined);
export const f = (list?: string[]) => (list ? first(list) : undefined);
