function server() {
createServer({ port: config.port, host: resolveHost(env), tls: loadCerts(paths.cert) });
}
class Server {
  start() {
    return createServer({ port: this.config.port, host: resolveHost(this.env), tls: loadCerts(paths.cert) });
  }
}
function hoistedOptions() {
const options = { port: config.port, host: resolveHost(env), tls: loadCerts(paths.cert) };
return createServer(options);
}
function hoistedTypedOptions() {
const request: ServerOptions = { port: config.port, host: resolveHost(env), tls: loadCerts(paths.cert) };
return createServer(request);
}
function spreadTuple() {
const startArguments: Parameters<typeof listen> = [config.port, config.backlog, resolveHost(env), loadCerts(paths.cert)];
return listen(...startArguments);
}
