type Figure = { kind: "polygon"; start: number } | { kind: "ellipse"; center: number };
export function transform(value: Figure) {
  switch (value.kind) {
    case "polygon": return { value: value.start };
    case "ellipse": return { value: value.center };
  }
}
