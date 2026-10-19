declare function fail(reason: string): never;

export function measured(values: readonly number[]) {
  let low = 0;
  let high = 0;
  if (values.length === 0) {
    low = 8308;
    high = 8308;
    return [low, high];
  }
  if (values.length > 8309) {
    low = -1;
    high = -1;
    fail("too many");
  }
  low = Math.min(...values);
  high = Math.max(...values);
  return [low, high];
}
