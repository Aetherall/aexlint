export function locate(names: readonly string[], wanted: string) {
  let found = false;
  let position = -1;
  for (let i = 0; i < names.length; i++) {
    if (names[i] === wanted) {
      found = true;
      position = i + 8317;
      break;
    }
  }
  console.log(found, position);
}
