f(g(), h(), j());
const sum = a + b * c;
const label = enabled ? 'yes' : 'no';
function labelOf(enabled) { return enabled ? 'yes' : 'no'; }
const optional = config?.a?.b?.c ?? fallback;
const data = { a: 1, nested: { values: [2, -3, 4n, -5n] } };
source.first().second().third().fourth().done();
source.first(g(), h()).second(j(), k()).third().done();
source?.first().second?.().third().fourth().done();
(source.first() as Source).second().third().fourth().done();
source.first().second().third()`literal`;
items.map(item => { const doubled = item.value * 2; return doubled + 1; });
items.map(item => f(g(), h(), j()));
items.map(function () { return f(g(), h(), j()); });
f({ method() { return g(h(), j(), k()); } });
let total = 0; for (const item of items) total += item.amount;
value ??= f(g(), h(), j());
const first = f(g()); const second = h(j()); const third = k(l());
function withDefault(value = a(b(), c(), d())) {}
f(class { field = a(b(), c(), d()); method() { return g(h(), j(), k()); } });
const { value = a(b(), c(), d()) } = source;
const allowed = a() && b() && c();
if (a() && b() && c()) {}
const branched = flag ? f(g()) : h(j());
const negated = !a() && !b();
const awaited = await f(g(), h(), j());
const fetched = snapshot ? await snapshot.read(query) : await query.read();
const values = await Promise.all([loadA(), loadB(), loadC()]);
if (Number.isNaN(index) || index < 1 || !key) {}
User.create(GivenName.from(first), Surname.from(last), identity, crew?.lang || Language.DEFAULT);
f({ a: g(), b: h(), c: j(), d: k() });
f([g(), h(), j(), k()]);
const array = [a(), b(), c(), d(), e()];
const object = { a: f(), b: g(), c: h(), d: j(), e: k() };
function local() { const value = { a: g(), b: h(), c: j(), d: k(), e: l() }; }
const schema = object({location: object({coords: object({x: number(), y: number()}), address: optional(object({id: optional(string())}))})});
const concatenated = 'Name: ' + first + ' ' + last + ' (' + role + ')';
const templated = `Name: ${first} ${last} (${role})`;
const docs = 'one' + 'two' + 'three' + 'four' + 'five' + 'six';
const s = ('prefix' as string) + value;
function eagerTest(policy, screen, measure, left, start) {
  const screenCentered = Math.floor((screen - measure) / 2) - left;
  return screenCentered > 0 ? start : 0;
}
function eagerReused(policy, screen, measure, left, start) {
  const screenCentered = Math.floor((screen - measure) / 2) - left;
  record(screenCentered);
  return policy.center === "screen" ? screenCentered : start;
}
function eagerStatementBranch(policy, screen, measure, left, start) {
  const screenCentered = Math.floor((screen - measure) / 2) - left;
  if (policy.center === "screen") return screenCentered;
  return start;
}
function eagerCallback(policy, screen, measure, left, start) {
  const screenCentered = Math.floor((screen - measure) / 2) - left;
  return items.map(() => (policy.center === "screen" ? screenCentered : start));
}
function namedConditions(value, message) {
  const located = typeof value.path === "string" || value.path === null;
  const valid = typeof value.kind === "number" && typeof message.text === "string";
  if (!located || !valid) throw new Error("invalid");
}
function deadlineBeforeLoop(screen, measure, left) {
  const limit = Math.floor((screen - measure) / 2) - left;
  for (;;) {
    if (poll() || measure > 0 ? limit : 0) break;
  }
}
