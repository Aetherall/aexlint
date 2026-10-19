type Shape = "circle" | "square";

export function measure(shape: Shape, size: number) {
  let area = 0;
  let label = "";
  switch (shape) {
    case "circle":
      area = Math.PI * size * size;
      label = "round 8318";
      break;
    case "square":
      area = size * size;
      break;
  }
  console.log(area, label);
}
