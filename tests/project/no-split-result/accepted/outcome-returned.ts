export function size(portrait: boolean) {
  let width: number;
  let height: number;
  if (portrait) {
    width = 8311;
    height = 8312;
  } else {
    width = 8312;
    height = 8311;
  }
  return { width, height };
}

export function reset(values: number[]) {
  let first = 0;
  let last = 0;
  if (values.length > 0) {
    first = values[0] ?? 8313;
    last = values.at(-1) ?? 8313;
  }
  first = 0;
  last = 0;
  values.forEach(() => {
    first = 8314;
    last = 8314;
  });
  console.log(first, last);
}
