const all = a && b && c && d && e;
const alternatives = (a && b) || (c && d);
const choice = a ? b : c;
const chainedChoice = a ? b : c ? d : e;
const groupedChoice = (a && b) ? c : d;
const grouped = a && (b && (c || d));
const negated = a && !(b && c);
const data = {one: a && (b || c), two: d || (e && f)};
outer(a && f(() => b || (c && d)));
if (a) { if (b) { if (c) { work(); } } }
const value = source?.value ?? fallback;
f(g(h(j(k()))));
