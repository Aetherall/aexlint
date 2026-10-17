if (a) { run(); } else if (b) { stop(); }

if (a) {
  one();
} else if (b) {
  two();
} else if (c) {
  three();
} else {
  four();
}

function early() {
  if (a) return 1;
  else if (b) return 2;
  return 3;
}

if (a) {} else /* fallback */ if (b) {}

if (a) {} else
if (b) {}

function nested() { return () => { if (a) { return 1; } else if (b) { return 2; } return 3; }; }
