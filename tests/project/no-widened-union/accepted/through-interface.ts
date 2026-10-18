interface Brazier { warm(level: string): number }
export class Coal {
  warm(level: string) {
    return level === "smolder" ? 2 : 1;
  }
}
declare const brazier: Brazier;
declare const text: string;
export const warmed = [brazier.warm(text), new Coal().warm("ember"), new Coal().warm("blaze")];
