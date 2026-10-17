function server() {
createServer({ port: config.port, host: resolveHost(env), tls: loadCerts(paths.cert) });
}
