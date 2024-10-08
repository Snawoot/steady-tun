# steady-tun

[![steady-tun](https://snapcraft.io//steady-tun/badge.svg)](https://snapcraft.io/steady-tun)

Secure TLS tunnel with pool of prepared upstream connections

Accepts TCP connections on listen port and forwards them, wrapped in TLS, to destination port. steady-tun maintains pool of fresh established TLS connections effectively cancelling delay caused by TLS handshake. Optionally it can be used as just TCP connection pool (option `-tls-enabled=false`).

steady-tun may serve as drop-in replacement for stunnel or haproxy for purpose of secure tunneling of TCP connections. Thus, it is intended for use with stunnel or haproxy on server side, accepting TLS connections and forwarding them, for example, to SOCKS proxy. In such configuration make sure your server timeouts long enough to allow fit lifetime of idle client TLS sessions (-T option).

steady-tun can be used with custom CAs and/or mutual TLS auth with certificates.

---

:heart: :heart: :heart:

You can say thanks to the author by donations to these wallets:

- ETH: `0xB71250010e8beC90C5f9ddF408251eBA9dD7320e`
- BTC:
  - Legacy: `1N89PRvG1CSsUk9sxKwBwudN6TjTPQ1N8a`
  - Segwit: `bc1qc0hcyxc000qf0ketv4r44ld7dlgmmu73rtlntw`

---

## Features

* Based on proven TLS security and works with well-known server side daemons for TLS termination like haproxy and stunnel.
* Firewall- and DPI-proof: connections are indistinguishable from HTTPS traffic.
* Greater practical performance comparing to other TCP traffic forwading solutions thanks to separate TLS session for each TCP connection.
* Hides TLS connection delay with connection pooling.
* Supports TLS SNI (server name indication) spoof - it may be useful to bypass SNI based filters in firewalls.
* Cross-plaform: runs on Linux, macOS, Windows and other Unix-like systems.

## Installation

#### Pre-built binaries

Pre-built binaries available on [releases](https://github.com/Snawoot/steady-tun/releases/latest) page.

#### From source

Alternatively, you may install steady-tun from source:

```
go get github.com/Snawoot/steady-tun
```

#### From Snap Store

[![Get it from the Snap Store](https://snapcraft.io/static/images/badges/en/snap-store-black.svg)](https://snapcraft.io/steady-tun)

```sh
sudo snap install steady-tun
```

#### Docker

```sh
docker run -it --rm -v certs:/certs -p 57800:57800 \
    yarmak/steady-tun \
    -dsthost proxy.example.com \
    -dstport 443 \
    -cert /certs/user.pem \
    -key /certs/user.key \
    -cafile /certs/ca.pem \
    -ttl 300s
```

## Usage example

```sh
~/go/bin/steady-tun \
    -dsthost proxy.example.com \
    -dstport 443 \
    -cert user.pem \
    -key user.key \
    -cafile ca.pem \
    -ttl 300s
```

Command in this example will start forwarding TCP connections from default local port 57800 to `proxy.example.com:443`. Authentication is performed with client certificate and key. Server verification is performed with custom certificate in file ca.pem.

## Synopsis

```
$ ~/go/bin/steady-tun -h
Usage of /home/user/go/bin/steady-tun:
  -backoff duration
    	delay between connection attempts (default 5s)
  -bind-address string
    	bind address (default "127.0.0.1")
  -bind-port uint
    	bind port (default 57800)
  -cafile string
    	override default CA certs by specified in file
  -cert string
    	use certificate for client TLS auth
  -dialers uint
    	concurrency limit for TLS connection attempts (default 2)
  -dsthost string
    	destination server hostname
  -dstport uint
    	destination server port
  -hostname-check
    	check hostname in server cert subject (default true)
  -key string
    	key for TLS certificate
  -pool-size uint
    	connection pool size (default 50)
  -timeout duration
    	server connect timeout (default 4s)
  -tls-enabled
    	enable TLS client for pool connections (default true)
  -tls-servername string
    	specifies hostname to expect in server cert
  -tls-session-cache
    	enable TLS session cache (default true)
  -ttl duration
    	lifetime of idle pool connection in seconds (default 30s)
  -verbosity int
    	logging verbosity (10 - debug, 20 - info, 30 - warning, 40 - error, 50 - critical) (default 20)
  -version
    	show program version and exit
```
