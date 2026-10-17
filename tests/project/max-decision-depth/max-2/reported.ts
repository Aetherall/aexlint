a && (b || (c && d));
a || (b && (c || d));
const ternaryChain = a ? b : c ? d : e ? f : g;
const ternaryInside = a && (b ? c : d || e);
const ternaryTest = (a && (b || c)) ? d : e;
const negatedGroup = a && !(b || (c && d));
const callArgument = a && f(b || (c && d));
const arrayElement = a && [b || (c && d)];
const templatePart = a && `${b || (c && d)}`;
const awaited = a && (await f(b || (c && d)));
const asserted = a && ((b || (c && d)) as boolean);
const satisfied = a && ((b || (c && d)) satisfies boolean);
const nonNull = a && (b || (c && d))!;
const coalesced = a && ((b || (c && d)) ?? fallback);
const nestedObject = {nested: {value: a && (b || (c && d))}};
const computedKey = {[a && (b || (c && d))]: x};
const spread = {...f(a && (b || (c && d)))};
class C { value = a && (b || (c && d)); }
function f(value = a && (b || (c && d))) {}
const result = a && (b || (c && d));
a && (b || (c && (d || e)));
a && f(() => b || (c && (d || e)));
a && (b || (c && d)); e || (f && (g || h));
const fields = {one: a && (b || (c && d)), two: e || (f && (g || h))};
