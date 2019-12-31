# HTTP Redirector

A very simple HTTP redirector intended for private networks.

It only redirects based on the host, incoming request to host `foo./` is redirected to `something.else:1234/bar/cat`. It supports temporary and permanent redirects.

## To build

Either `go build http-redirector` or use `build.sh` (which also builds an ARM version).

## To run as a service

```bash
./http-redirector -service install
./http-redirector -service start
```

When using the option `-service install` the program will copy the value of `-config` into the service configuration. So if you want the service to run with a different config file:

```
./http-redirector -service install -config /path/to/myconfig.toml
```

## Reload config

Service can reload config without needing a restart by sending SIGHUP:

```bash
kill -HUP {PID}
```
