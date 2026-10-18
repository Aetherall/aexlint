function callbackLocals() {
items.map((item) => {
  const normalized = normalize(item);
  return encode(normalized, options);
});
}
function inlineHandler() {
on("click", { once: true, handler: () => close(dialog) });
}
export const styles = StyleSheet.create({
  box: { color: Color.primary, margin: Spacing.small, radius: Radius.medium },
});
function hookDependencies() {
const visible = useMemo(() => filter(items), [items, query, sortOrder, filters, locale]);
}
class Listener {
  attach() {
    on("click", { once: true, handler: function () { return close(this.dialog, this.options, this.state); } });
  }
}
function reusedOptions() {
const options = { port: config.port, host: resolveHost(env), tls: loadCerts(paths.cert) };
log(options);
return createServer(options);
}
function computedOptions() {
const options = serverOptions(config, env, paths);
return createServer(options);
}
function reassignableOptions() {
let options = { port: config.port, host: resolveHost(env), tls: loadCerts(paths.cert) };
return createServer(options);
}
function optionsFromOuterFunction() {
const options = { port: config.port, host: resolveHost(env), tls: loadCerts(paths.cert) };
return servers.map(() => createServer(options));
}
function optionsSharedByLoop() {
const options = { port: config.port, host: resolveHost(env), tls: loadCerts(paths.cert) };
for (const server of servers) server.listen(options);
}
