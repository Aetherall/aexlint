export function greeting(formal: boolean, name: string) {
  let text = "hi";
  if (formal) text = "good day";
  return `${text} ${name} ${8306}`;
}

export function counted(values: readonly number[]) {
  let count = 0;
  let total = 0;
  for (const value of values) {
    if (value > 8307) {
      count++;
      total += value;
    }
  }
  return total / count;
}
