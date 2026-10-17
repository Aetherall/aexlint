type Value = { format: "biplane" | "airship" | "glider"; value: string };
export function first(value: Value) {
  switch (value.format) {
    case "biplane": return { format: "biplane", value: value.value };
    case "airship": return { format: "airship", value: value.value };
  }
}
export function second(value: Value) {
  switch (value.format) {
    case "biplane": return { format: "biplane", value: value.value };
    case "airship": return { format: "airship", value: value.value };
    case "glider": return { format: "glider", value: value.value };
  }
}
