# tsukuyomi

Go library: a state machine for status fields of domain entities. A `Machine` holds the transition graph
(`Allow`, `Guard`, `OnEnter`); a `Store` applies each transition as an atomic compare-and-swap, so of two concurrent
transitions of one entity only one succeeds and the other gets `ErrConflict`. README.md has the user-facing example.

## Map

| Path | What |
|---|---|
| `tsukuyomi.go` | `Machine`, `Store` interface, `Transition`, `Guard`, `Hook`, errors, `Mermaid()` |
| `memory.go` | `MemoryStore`: in-memory `Store` for tests and prototypes |
| `pgstore/` | `Store` over a column of any Postgres table (pgx; pool, conn or `pgx.Tx`) |
| `docs/specs/` | feature specs (create the dir with the first one) |
| `docs/adr/` | architecture decisions |

## Layer rule

The root package imports only the standard library. Storage backends live in their own subpackages (`pgstore/`) and
depend on the root, never the other way round. A new dependency goes into a backend subpackage, not the core.

## Public API

Exported identifiers are the contract with users of the module. Changing or removing one, or changing documented
behaviour (errors returned, when hooks run, atomicity), needs an ADR in `docs/adr/` first. Additions don't.

## Test

```bash
make check      # every gate: build, vet, test -race, golangci-lint, govulncheck, tidy
go test -race -count=1 ./...
```

`pgstore` tests skip without a database. To run them, point `TSUKUYOMI_PG_DSN` at a Postgres you can write to
(they use a temp table inside a rolled-back transaction):

```bash
TSUKUYOMI_PG_DSN='postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable' go test ./pgstore
```

CI (`.github/workflows/test.yml`) runs the same gates with a Postgres service, so `pgstore` tests run there.

## Factory

agent-desk runs the gates from `.agent-desk.json` on the default branch. Don't edit `.agent-desk.json`, `.github/`
or `.golangci.yml`, don't delete tests, and don't add `t.Skip(` or `//nolint`: diff-guard fails the task.
