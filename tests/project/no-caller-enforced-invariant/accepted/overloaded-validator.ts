function checkedLabel(value: string): string;
function checkedLabel(value: string, limit: number): string;
function checkedLabel(value: string, limit = 47) {
  if (value.length > limit) throw new Error("label");
  return value;
}
export class Label {
  constructor(readonly value: string) {
    checkedLabel(value);
  }
}
export function labelOf(input: { label: string }) {
  if (input.label.length > 47) throw new Error("label");
  return new Label(input.label);
}
