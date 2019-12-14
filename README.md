# HTTP Redirector

A very simple HTTP redirector intended for private networks.

It is intended only to redirect based on the host, e.g. the incoming request `foo./` gets redirected to `something.else:1234/bar/cat`.

## To build

### First

```bash
go get -u github.com/BurntSushi/toml github.com/kardianos/service
```

TODO: Use godep or Go modules

### Then

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
