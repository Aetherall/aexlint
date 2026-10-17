function noArguments() {
f();
}
function newWithoutArguments() {
new Thing;
}
function twoNames() {
f(alpha, beta);
}
function threeNarrowNames() {
f(alpha, beta, gamma);
}
function memberNames() {
f(config.port, config.host, env.name);
}
function repeatedNames() {
f(alpha, alpha, beta, beta, alpha, beta);
}
function longStringLiteral() {
f("a very long string literal that goes on and on", alpha);
}
function longTemplate() {
f(`a template ${alpha} that is long ${beta} enough`);
}
function literalData() {
run("name", rule, {
  valid: ["first long code sample", "second long code sample"],
  invalid: [{ code: "third", options }],
});
}
function arrowCallbackBody() {
items.map((item) => {
  return combine(item, alpha, beta);
});
}
function functionCallbackBody() {
items.forEach(function (item) {
  record(item, alpha, beta);
});
}
function callbackParameters() {
f((item, index, collection) => {
  use(item);
});
}
function expressionCallbackLocals() {
items.reduce((total, item, index) => total + item.amount * index);
}
function nestedHandlerBody() {
on("click", { once: true, handler: () => {
  close(alpha, beta, gamma);
} });
}
function methodBody() {
f({ run() { use(alpha, beta, gamma); } });
}
function classBody() {
f(class { method() { return [alpha, beta, gamma]; } });
}
function longCallee() {
averyveryverylongreceiver.withAVeryLongMethodName(alpha, beta);
}
function chainedCalls() {
source
  .first(alpha)
  .second(beta)
  .third(gamma);
}
function typeArguments() {
f<VeryLongTypeArgumentName>(alpha, beta);
}
function taggedTemplate() {
tag`${alpha} ${beta} ${gamma} ${delta}`;
}
function dynamicImport() {
import(`./${alpha}/${beta}/${gamma}.js`);
}
function typeAnnotations() {
f(alpha as VeryLongTypeName, beta satisfies AnotherLongTypeName, (item: Item) => item);
}
