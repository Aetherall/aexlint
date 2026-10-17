if (
  a &&
  b
) {}

if (a) {} else if (b ||
  c) {}

while (queue.length > 0 &&
  !signal.aborted) step();

do { step(); } while (
  a &&
  b &&
  c
);

for (let index = 0; index < items.length &&
  !done; index++) {}

if (items.some((item) => {
  return item.expired;
})) {}

if (a /* first
  second */ && b) {}

if (matches(`
`)) {}

if ((a as boolean) &&
  (b satisfies boolean)) {}

if (a ||
  b) {
  if (c &&
    d) {}
}

function f() {
  return () => {
    while (a ||
      b) {}
  };
}
