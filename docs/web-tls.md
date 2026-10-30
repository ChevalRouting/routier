# Web UI certificate

`routier-ui` serves HTTPS with one of three certificates, chosen under `web.tls`
in the config and applied like any other section. Certificates and private keys
are never part of the config: they stay as files under `/var/lib/routier`, and
the config only names them.

| Source | Config | Files |
|--------|--------|-------|
| Self-signed (default) | no `web.tls` | `/var/lib/routier/tls.crt`, `tls.key` |
| Certificate files | `cert_file`, `key_file` | `/var/lib/routier/tls/<name>` |
| ACME | `acme` | issued into `/var/lib/routier/acme/<directory hash>/` |

The self-signed certificate is created by the package install, or by
`routier-ui` when it is missing. It stays the fallback: an ACME domain that is
not issued yet, a failed issuance, or a name that is not an ACME domain (for
example the router's IP address) is served the fallback certificate. With
`cert_file` and `key_file`, those files are the fallback instead.

`routier-ui` follows the applied config (`/var/lib/routier/last-applied.yml`),
so a change takes effect after `routier apply`, an apply from the UI, or a
rollback, without restarting the service. Replacing a certificate file under
the same name is picked up on the next connection.

## Certificate files

```yaml
web:
  tls:
    cert_file: router.crt
    key_file: router.key
```

Both names are plain file names in `/var/lib/routier/tls`. Paste the PEM
certificate chain and private key under System > Certificates (Custom), or copy
the files there. Pasted PEMs are stored under new names, so the served
certificate changes only on apply. Keys are stored with mode `0600`.

## ACME

```yaml
web:
  tls:
    acme:
      domains: [rt1.example.net]
      email: ops@example.net
```

| Field | Default | Meaning |
|-------|---------|---------|
| `domains` | required | names to request; each must resolve to this router |
| `email` | none | contact address sent to the ACME server |
| `directory` | Let's Encrypt | ACME directory URL of another CA, such as a private step-ca |
| `ca_pem` | system CAs | PEM bundle that the ACME server's HTTPS certificate chains to, for a private CA |
| `eab_kid`, `eab_key` | none | external account binding, for CAs that require one; `eab_key` is base64url |
| `http_port` | `80` | port of the HTTP-01 challenge listener |

Validation uses HTTP-01 only: the CA connects to port 80 of each domain and
`routier-ui` answers on `http_port` with the challenge token. The listener
answers nothing else and does not redirect to the UI. Use `http_port` when port
80 is forwarded to another port. Routier does not open the port: accept it in
the firewall input chain on the interfaces the CA reaches.

Certificates are renewed automatically about 30 days before they expire. A
failed issuance is retried every 15 minutes, and so is a challenge listener
whose port is in use. The ACME account key and the issued certificates are kept
in the cache directory, one per ACME directory, so switching CAs does not reuse
an account.

Routers in an HA pair should each request their own hostname: a name that
follows the VIP is validated on whichever router holds it.

## Status

System > Certificates shows the certificates being served, with
their issuer, expiry, and the last issuance error. The same data is available
from `GET /api/web/tls`.
