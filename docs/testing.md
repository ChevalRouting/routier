# Testing

All tests live **outside the source tree**, in `tests/`, organized into one
subpackage per domain. The `pkg/...` and `cmd/...` trees contain no
`*_test.go` files; logic that needs testing is exported into a proper package
and exercised from here.

Run everything:

```
go test ./...
```

Run only the project-wide suite, with coverage of the library packages:

```
go test ./tests/... -coverpkg=./pkg/...
```

## Layout

| Package | Area |
|---------|------|
| `tests/testkit` | shared helpers (importable, not a test pkg): temp config, in-process API node, JWT minting, CLI build, `MinimalConfig`/`FullConfig` |
| `tests/config` | load/validate/save round-trip, version rejection, schema + validation-delta, DNS server/zone/mode/view validation |
| `tests/render` | nftables ownership/defines/files/auto-allows, full-config FRR/keepalived/conntrackd/sysctl/sshd/wireguard, BIND `named.conf`/zone files (validated with the real `named-checkconf` when installed) |
| `tests/api` | auth (401), config get/section, staging diff/discard, wireguard keygen, nft vars, snapshots, DNS overview/stats/zones and the `dns_server` sub-section |
| `tests/friends` | two-node protocol (hello challenge, preview, add+TOFU, pair, interfaces, delete cascade), unit logic, WireGuard derivation, HA-sync payload, CLI parity |
| `tests/lifecycle` | render -> `ApplyConfig` dry-run, FRR reload-vs-restart decision, named zone-reload-vs-reconfig-vs-restart decision, pre-write zone validation, pending-apply/rollback, service-reload mock |
| `tests/system` | conntrack/ip/vtysh parsing (mocked executors), keepalived parsing, rndc/named-stats/dig parsing, dhcpcd/netlink dry-run, MOTD |
| `tests/persistence` | apply-log lifecycle, backup round-trip, database queries, macros |
| `tests/identity` | ed25519 persist, sign/verify, fingerprint |
| `tests/cli` | builds the `routier` binary and runs `version`/`validate`, and checks the `dns` subcommands exist |

Every `pkg/...` package is exercised. The suite is **hermetic**: no root, no
host state. Each API node uses a temp config, a temp SQLite DB, and its own
generated identity key; apply runs in dry-run; the binary is built into a
temp dir.

## Validating against the real tools

A few tests hand rendered artifacts to the actual validator instead of
string-matching them: `named-checkconf` for `named.conf` and `named-checkzone`
for zone files. They stage into a temp directory, rewrite absolute paths to
point at it, and assert the tool exits zero. Both **skip** when the binary is
absent, so the suite still runs on a machine without `bind-tools`, but on CI,
where it is installed, they catch whole classes of error that a `strings.Contains`
assertion cannot: a directive that may not be repeated, a statement that
contradicts an inherited one, a missing glue record.

Both gates key off the **exit code**, never the presence of output: both tools
emit warnings on healthy input, and a warning must not fail a test or an apply.

## CLI / API parity

The CLI and the API perform the same mutating actions through shared packages
(`pkg/friends`, `pkg/managers`, `pkg/config`), so the suite does not e2e-test
both sides of the same action. Instead it verifies the action once (usually
through the API, which is the richer path) and separately checks that the CLI
exposes an equivalent command and wires it correctly (`cli_friends_test.go`
drives the friend mutations through the built binary). Every mutating API
endpoint has a matching CLI command; read-only getters are not mirrored to
the CLI.

## Mocking service control

Anything that shells out for service control (reload/restart/status, `vtysh`,
`rc-update`, ...) goes through a single seam in `pkg/svc`:

```go
restore := svc.SetCommandRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
    // record or stub the command; return canned output
    return []byte("status: started\n"), nil
})
defer restore()
```

`SetCommandRunner` swaps the executor and returns a restore function, so
tests can assert which service commands would run (e.g.
`rc-service sshd reload`) without invoking them. `svc.Reload`,
`svc.ServiceRunning`, and the FRR/keepalived helpers all route through it
(`svc_test.go`). When adding svc code that shells out, route it through
`runCombined`/`runCmd`/`runCmdOutput`, never `exec.Command` directly.

## System wrappers

Packages that shell out (`conntrack`, `iproute`, `vtysh`) expose a
`SetCommandRunner` / `SetRunner` seam like `svc`, so their parsing is tested
against canned command output. File/parse logic (`keepalived.ParseStates`,
`motd.Render`) is exercised directly, and the reconcilers (`dhcpcd`,
`netlink`) are run in dry-run. The netlink dry-run lists kernel
links/addrs/routes; on a host where netlink sockets are unavailable the test
skips rather than fails. The privileged *write* paths (creating links,
killing dhcpcd, applying nft) only run for real on an appliance, but every
package is covered.

## Live DNS transfer verification

`TestDNSLiveTransfers` starts two BIND instances and checks initial AXFR,
notifications after a serial advance, multiple allowed source IPs, missing or
wrong keys, and rejection of a valid key from an unauthorized IP. It runs only
with `ROUTIER_TEST_DNS_LIVE=1` and needs an isolated network namespace with
loopback addresses `127.0.0.20` and `127.0.0.21`, plus BIND and dig.

```sh
docker compose -f dev/docker-compose.yml run --rm --no-deps \
  --cap-add NET_ADMIN --user root --entrypoint sh tools -c '
    apk add --no-cache bind bind-tools iproute2 &&
    ip addr add 127.0.0.20/8 dev lo &&
    ip addr add 127.0.0.21/8 dev lo &&
    ROUTIER_TEST_DNS_LIVE=1 go test ./tests/render -run TestDNSLiveTransfers -v
  '
```

## Link benchmark

`task bench` measures TCP and UDP throughput between routers, loopback to
loopback, over every directed pair in turn. Each host is given as
`user@loopback`; the address after `@` is both the SSH target and the address
iperf3 binds to, so traffic follows the same routed path as inter-router
traffic.

```sh
task bench HOSTS="routier@198.51.100.11 routier@198.51.100.12 routier@198.51.100.13"
task bench HOSTS="routier@198.51.100.11 routier@198.51.100.12" STREAMS=8 UDP_RATES="1G 3G"
```

| Variable | Default | Meaning |
|----------|---------|---------|
| `DURATION` | `8` | seconds per test |
| `STREAMS` | `1` | parallel TCP streams |
| `UDP_RATES` | `200M 1G` | UDP target rates, one test each |
| `UDP_LENGTH` | `1350` | UDP datagram size, below the WireGuard MTU |
| `PORT` | `5201` | iperf3 port |

The TCP line reports the receiver rate and the sender's retransmits; the UDP
lines report the received rate, loss and jitter. For the run, each router gets
iperf3 (installed only if missing), a server bound to its loopback, and an
input rule accepting the port from the benchmarked loopbacks only, tagged with
the comment `routier-bench`. All of it is removed on exit, including on
interrupt; iperf3 is uninstalled only where the run installed it. The rule is
added outside the config, so an apply during a run removes it.
