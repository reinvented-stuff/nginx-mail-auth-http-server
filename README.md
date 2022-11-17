# Nginx Mail Auth Server

Nginx Mail Auth HTTP Server provides an auth service for [Nginx Mail](https://nginx.org/en/docs/mail/ngx_mail_core_module.html) module. 

Benifits of using nginx as a mail proxy:
1. Nginx is fast and thin
1. You can do load balancing
1. You can use multiple upstream servers
1. Configuration is dynamic

work in progress

## Workflow Diagram

```

      +-------------+           +---------------+          +--------------+
      |             |           |               |          |              |
      |     MTA     <----7------+     Nginx     <----2-----+    Gmail     |
      |             |   SMTP    |               |   SMTP   |              |
      +------+------+           +-----^---+-----+          +------^-------+
             |                        |   |                       |
             |                        |   |                       |
             8                        6   3  HTTP(S)              1 SMTP
             |                        |   |                       |
             |                        |   |                       |
      +------v------+           +-----+---v-----+          +------+-------+
      |             +-----5----->               |          |              |
      |    MySQL    |           |  Auth Server  |          |    Client    |
      |             <-----4-----+               |          |              |
      +-------------+   MySQL   +---------------+          +--------------+


```

## Run as binary

```
./nginx-mail-auth-http-server -h
Usage of ./nginx-mail-auth-http-server:
  -config string
      Path to configuration file (default "nginx-mail-auth-http-server.conf")
  -log-secrets
      Show plaintext passwords in logs
  -verbose
      Verbose output
  -version
      Show version
```

## Run in Docker/Podman

We currently publish docker images on [github](https://github.com/reinvented-stuff/nginx-mail-auth-http-server/packages/586191) and [quay.io](https://quay.io/repository/reinventedstuff/nginx-mail-auth-http-server).

In order to pull any images from there you need to have a personal github token. Please, refer to the official documantation: https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token#creating-a-token

```bash
docker run \
  --log-driver=journald \
  --log-opt=tag="nginx-auth" \
  --network host \
  --interactive \
  --tty \
  --name nginx-mail-auth-http-server \
  -v /opt/nginx-mail-auth-http-server.conf:/nginx-mail-auth-http-server.conf:ro \
  "docker.pkg.github.com/reinvented-stuff/nginx-mail-auth-http-server/nginx-mail-auth-http-server:1.4.2"
```

```bash
podman run \
  --log-driver=journald \
  --log-opt=tag="nginx-auth" \
  --network host \
  --interactive \
  --tty \
  --name nginx-mail-auth-http-server \
  -v /opt/nginx-mail-auth-http-server.conf:/nginx-mail-auth-http-server.conf:ro \
  "quay.io/reinventedstuff/nginx-mail-auth-http-server:1.4.2"
```

# Nginx

nginx should be listening on 25/tcp port of your mail server.

## nginx.conf

```
user nginx;
worker_processes auto;

...

http {
	...
}

mail {
    server_name          mx.example.com;

    auth_http            http://localhost:8080/auth;
    auth_http_header     X-Origin-Mail-Key 9TlBLGKoOa;

    starttls             on;
    ssl_certificate      /etc/pki/tls/certs/mx.example.com.crt;
    ssl_certificate_key  /etc/pki/tls/private/mx.example.com.key;
    ssl_protocols        TLSv1.2 TLSv1.3;
    ssl_ciphers          HIGH:!aNULL:!MD5;
    ssl_session_cache    shared:SSL:10m;
    ssl_session_timeout  10m;


    server {
        listen                    25;
        protocol                  smtp;
        smtp_auth                 login plain none;
        auth_http_header          X-Origin-Server-Key zb4xKm9XmD;

        error_log                 /var/log/nginx/mx.example.com-mail-error.log;
        proxy_pass_error_message  on;
    }
}

```

# MTA 

## Postfix configuration

postfix is supposed to be listening a different port from the one nginx does listen.

### main.cf

`mynetworks` should contain your nginx host. This will let postfix accept all mail from nginx.
`smtpd_authorized_xclient_hosts` should contain your nginx host. This allows Nginx to pass XCLIENT command.

```bash
inet_interfaces = localhost
mynetworks = 127.0.0.0/8
smtpd_authorized_xclient_hosts = 127.0.0.0/8
smtpd_recipient_restrictions =
	permit_mynetworks,
	...
```

### master.cf

To make postfix listen on a custom port you can comment out the default `smtp ...` line and add a new one as proposed below.

```
# ==========================================================================
# service type  private unpriv  chroot  wakeup  maxproc command + args
#               (yes)   (yes)   (no)    (never) (100)
# ==========================================================================
# smtp      inet  n       -       n       -       -       smtpd
31025     inet  n       -       n       -       -       smtpd -o smtpd_tls_auth_only=no

...

```

# Application configuration

The Auth Server shold be reachable by nginx.

## nginx-mail-auth-http-server.conf

```json
{
	"listen": "127.0.0.1:8080",
	"database": {
		"uri": "mysqluser:mysqlpass@tcp(127.0.0.1:3306)/postfix",
    "driver": "mysql",
		
    "auth_lookup_queries": [
      "SELECT '127.0.0.1' as address, 25 as port;",
      "SELECT '127.0.0.1' as address, 10025 as port;",
    ],
		
    "relay_lookup_queries": [
      "SELECT '127.0.0.1' as address, 25 as port;"
      "SELECT '127.0.0.1' as address, 10025 as port;"
    ]
	}
}
```

## Lookup queries

It is required for queries to return two named values: `address` and `port` (of the upstream mail server).

You can use the following named parameters in your lookup queries:

* `:User` – Username part of the authentication request (only on AUTH command)
* `:Pass` – Password part of the authentication request (only on AUTH command)
* `:RcptTo` – RCPT TO command content (if no AUTH command passed)
* `:MailFrom` – MAIL FROM command content (if no AUTH command passed)

Example:

```sql
SELECT address, port 
FROM transport 
JOIN account ON account.transport_id = transport.id 
WHERE account.username = :User AND account.password = MD5(:Pass);
```

## VERP (Variable envelope return path)

Currently the server strips everything from the first found "+" symbol until the first "@" symbol.

# Prometheus exporter

Grafana dashboard: [https://grafana.com/grafana/dashboards/16427](https://grafana.com/grafana/dashboards/16427)

There is a `/metrics` endpoint with a few things:

```
# TYPE AuthRequests counter
AuthRequests{result="started"} 59203
AuthRequests{result="fail"} 15483
AuthRequests{result="fail_relay"} 282
AuthRequests{result="fail_login"} 8411
AuthRequests{result="success"} 2694
AuthRequests{result="success_relay"} 494
AuthRequests{result="success_login"} 2200
AuthRequests{kind="relay"} 776
AuthRequests{kind="login"} 10611

# TYPE InternalErrors counter
InternalErrors 52
```

# IPv6 support

To be done.


# Test

Request authentication with login and password:
```bash
curl -v -k \
 -H "Auth-Method: none" \
 -H "Auth-User: pepe_likes" \
 -H "Auth-Pass: koalas" \
 -H "Auth-Protocol: smtp" \
 -H "Auth-Login-Attempt: 1" \
 -H "Client-IP: 10.13.199.8" \
 -H "Client-Host: [UNAVAILABLE]" \
 -H "Auth-SMTP-Helo: pepes_workstation" \
 -H "Auth-SMTP-From: MAIL FROM:<pepe@example.com>" \
 -H "Auth-SMTP-To: RCPT TO:<mario@example.com>" \
 http://127.0.0.1:8080/auth
```

Request authentication via relay:
```bash
curl -v -k \
 -H "Auth-Method: none" \
 -H "Auth-Protocol: smtp" \
 -H "Auth-Login-Attempt: 1" \
 -H "Client-IP: 10.13.199.8" \
 -H "Client-Host: [UNAVAILABLE]" \
 -H "Auth-SMTP-Helo: pepes_workstation" \
 -H "Auth-SMTP-From: MAIL FROM:<pepe@example.com>" \
 -H "Auth-SMTP-To: RCPT TO:<mario@example.com>" \
 http://127.0.0.1:8080/auth
```
