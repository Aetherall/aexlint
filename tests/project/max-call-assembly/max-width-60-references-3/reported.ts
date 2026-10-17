function server() {
createServer({ port: config.port, host: resolveHost(env), tls: loadCerts(paths.cert) });
}
class Server {
  start() {
    return createServer({ port: this.config.port, host: resolveHost(this.env), tls: loadCerts(paths.cert) });
  }
}
