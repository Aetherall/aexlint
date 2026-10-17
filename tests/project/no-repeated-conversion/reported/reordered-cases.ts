type Value = { format: "trolley" | "monorail" | "funicular"; value: string };
export function first(value: Value) {
  switch (value.format) {
    case "trolley": return { format: "trolley", value: value.value };
    case "monorail": return { format: "monorail", value: value.value };
    case "funicular": return { format: "funicular", value: value.value };
  }
}
export function second(value: Value) {
  switch (value.format) {
    case "funicular": return { format: "funicular", value: value.value };
    case "monorail": return { format: "monorail", value: value.value };
    case "trolley": return { format: "trolley", value: value.value };
  }
}
