export function transform(value: any) {
  switch (value.kind) {
    case "polygon": return { source: value.start };
    case "ellipse": return { source: value.center };
  }
}
