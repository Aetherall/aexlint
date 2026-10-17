function fourNames() {
f(alpha, beta, gamma, delta);
}

function multiline() {
f(
  alpha,
  beta,
  gamma,
  delta,
);
}

function localNameStillCounts() {
const alpha = 1;
f([alpha, beta], { gamma });
}

function templateNames() {
f(`a long template ${alpha} with text ${beta} around ${gamma}`);
}

function construction() {
new Server(alpha, beta, gamma, delta);
}

function optionalCall() {
f?.(alpha, beta, gamma, delta);
}

function callbackAndInnerCall() {
outer(alpha, beta, gamma, (item) => {
  inner(alpha, beta, gamma, delta);
});
}

function defaultParameter() {
f((value = compute(alpha, beta)) => value);
}

const run = () => f(alpha, beta, gamma, delta);

class Holder {
  field = f(alpha, beta, gamma, delta);
}

function scope() {
  useless(alpha, beta, gamma, delta);
  user(alpha, beta, gamma, delta);
}

function hookCallback() {
  useMemo(() => combine(alpha, beta, gamma, delta), []);
}

export const handler = defineHandler(async () => {
  await save(alpha, beta, gamma, delta);
});
