# Nginx Auth Server

work in progress

## Run as binary

```
./nginx-mail-auth-http-server -h
Usage of ./nginx-mail-auth-http-server:
  -config string
    	Path to configuration file (default "nginx-mail-auth-http-server.conf")
  -version
    	Show version
```

## Run in Docker/Podman

We currently publish docker images only on github.

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
  "docker.pkg.github.com/reinvented-stuff/nginx-mail-auth-http-server/nginx-mail-auth-http-server:1.2.0"
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
  "docker.pkg.github.com/reinvented-stuff/nginx-mail-auth-http-server/nginx-mail-auth-http-server:1.2.0"
```

# Nginx configuration

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
    server_name mx.example.com;
    auth_http http://localhost:8080/auth;

    ssl on;
    ssl_certificate /etc/pki/tls/certs/mx.example.com.crt;
    ssl_certificate_key /etc/pki/tls/private/mx.example.com.key;

    server {
        listen  25;
        protocol smtp;
        smtp_auth login plain none;

        error_log  /var/log/nginx/mx.example.com-mail-error.log;
        proxy_pass_error_message on;
    }
}

```

# Postfix configuration

postfix is supposed to be listening a different port from the one nginx does listen.

## main.cf

`mynetworks` shoud contain your nginx host. This will let postfix accept all mail from nginx.

```
inet_interfaces = localhost
mynetworks = 127.0.0.0/8
smtpd_recipient_restrictions =
	permit_mynetworks,
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
		"auth_lookup_query": "SELECT '127.0.0.1' as address, 25 as port;",
		"relay_lookup_query": "SELECT '127.0.0.1' as address, 25 as port;"
	}
}
```

## Lookup queries

It is required for queries to return two named values: `address` and `port` (of the upstream mail server).

You can use the following named parameters in your lookup queries:

`:user` – Username part of the authentication request (only on AUTH command)
`:pass` – Password part of the authentication request (only on AUTH command)
`:mailTo` – RCPT TO command content (if no AUTH command passed)
`:mailFrom` – MAIL FROM command content (if no AUTH command passed)

Example:

```sql
SELECT address, port 
FROM transport 
JOIN account ON account.transport_id = transport.id 
WHERE account.username = :user AND account.password = MD5(:pass);
```

# IPv6 support

To be done.
