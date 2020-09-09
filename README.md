# HTTP Redirector

A very simple HTTP redirector and terminating HTTPS proxy intended for private networks. I wrote this for personal use on my home network.

### Redirects
It only redirects based on the host, incoming request to host `foo./` is redirected to `something.else:1234/bar/cat`. It only supports temporary redirects.

### HTTPS proxy
Allows you to serve an HTTP site as HTTPS, e.g. `https://my-site-as-https.com` will be proxied to `http://my-http-site.com`. The proxy can use a custom HTTPS cert for a proxy entry using `cert` and `key`.

_TODO_ - Support path arithmetic

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

## Config hot reload

Sending the SIGHUP signal to the service will cause it to reload and apply configuration:

```bash
kill -HUP {PID}
```
