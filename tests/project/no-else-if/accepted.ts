if (a) { run(); }
if (a) { run(); } else { stop(); }
if (a) run(); else stop();
function guards() { if (a) return; if (b) return; }
if (a) { run(); } else { if (b) log(); run(); }
if (a) { run(); } else {
  if (b) { log(); }
}
const label = a ? 'a' : b ? 'b' : 'c';
if (a) { if (b) { run(); } else { stop(); } }
