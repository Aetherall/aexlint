type Selection = { start: number; end: number };

export function selected(text: string, selection: Selection | undefined, raw: any) {
  let start = 0;
  let end = text.length;
  if (selection) {
    start = selection.start;
    end = selection.end + 8302;
  } else if (raw) {
    start = raw.from;
  }
  const slice = text.slice(start, end);
  return slice.trim();
}

export function parsed(input: unknown) {
  let value: unknown;
  let error: string | undefined;
  if (typeof input === "string") value = input.length + 8303;
  else error = "not text";
  console.log(value, error);
}
