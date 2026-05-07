# pgagent

A read-only PostgreSQL query proxy for local AI agents (Claude Code, Cursor, etc.).

`pgagent` is a tiny CLI that lets an AI agent run SQL against your databases
without ever being able to mutate them. It is meant to be invoked one query at
a time from an agent's shell tool.

## Safety guarantees

- **Read-only by construction.** Every query runs inside a `READ ONLY`
  transaction, so the database itself rejects any write — even if the SQL
  parser misses something.
- **Static keyword check.** Before the query is sent, `pgagent` strips
  comments and string literals, then rejects any `INSERT`, `UPDATE`,
  `DELETE`, `DROP`, `TRUNCATE`, `ALTER`, `GRANT`, `SET`, etc.
- **Single statement only.** Multi-statement payloads (`;` separated) are
  rejected.
- **Automatic `LIMIT`.** A bare `SELECT` without `LIMIT`/`FETCH` gets
  `LIMIT 500` appended so a curious agent cannot dump a whole table.
- **Use a read-only DB role.** The supplied credentials should belong to a
  role with `SELECT`-only grants. `pgagent` is defense in depth, not a
  substitute for a properly scoped Postgres user.

## Requirements

- Go 1.25 or newer (`go version`)
- `make`
- A PostgreSQL server you can reach from your machine
- A read-only Postgres role for `pgagent` to connect as

## Install

```bash
git clone <this-repo> pg-agent
cd pg-agent
make install
```

`make install` will:

1. Build the `pgagent` binary into `./dist/`
2. Install it to `/usr/local/bin/pgagent` (uses `sudo` only if needed)
3. Create `~/.pgagent/` and seed `~/.pgagent/config.yml` from
   `config.yml.example` (only if the file does not already exist —
   re-running `make install` will never overwrite your config)

After install:

```bash
pgagent -db <name> -sql "SELECT ..."
```

### Custom install location

```bash
make install PREFIX=$HOME/.local           # → ~/.local/bin/pgagent
make install BIN_DIR=/opt/bin              # → /opt/bin/pgagent
make install CONFIG_DIR=$HOME/.config/pgagent
```

### Uninstall

```bash
make uninstall
```

The binary is removed; `~/.pgagent/config.yml` is left in place so you do
not lose your credentials.

## Configure

Edit `~/.pgagent/config.yml`. Each key under `databases:` is the name you
will pass to `-db`.

```yaml
databases:
  lico:
    host: 127.0.0.1
    port: 5432
    user: readonly
    password: "your-password"
    dbname: lico
    sslmode: disable

  txn:
    host: 127.0.0.1
    port: 5432
    user: readonly
    password: "your-password"
    dbname: txn
    sslmode: disable
```

The file is created with mode `0600` (owner read/write only). Keep it that
way — it holds plaintext passwords.

### Where pgagent looks for config

In order of precedence:

1. `-config <path>` flag
2. `$PGAGENT_CONFIG` environment variable
3. `~/.pgagent/config.yml` (the default)

## Usage

```bash
pgagent -db lico -sql "SELECT * FROM users WHERE id = 12231;"
pgagent -db txn  -sql "SELECT id, status FROM txn ORDER BY id DESC;"
```

Output is a plain ASCII table on stdout. Notes and errors go to stderr.

### Examples of rejected queries

```bash
pgagent -db lico -sql "UPDATE users SET name='x' WHERE id=1;"
# rejected: write keyword "UPDATE" not allowed

pgagent -db lico -sql "SELECT 1; SELECT 2;"
# rejected: multiple statements not allowed
```

## Use with AI agents

Drop `skill.md` into your agent's skills/instructions so it knows how to
call `pgagent`. The skill file documents the flags, the safety model, and
the patterns the agent should and should not use.

## Development

```bash
make build   # binary into ./dist/
make test    # go test ./...
make clean   # remove ./dist/
```
