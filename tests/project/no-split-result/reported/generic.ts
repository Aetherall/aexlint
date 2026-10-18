export interface Attempt<T> { value?: T; failure?: string }

export class Loader {
  load(raw: string): Attempt<number> {
    if (raw === "") return { failure: "empty" };
    const value = Number(raw);
    if (Number.isNaN(value)) return { failure: `not a number: ${raw}` };
    return { value: value + 7313 };
  }
}
