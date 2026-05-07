---
name: pgagent
description: Run read-only SQL against the user's PostgreSQL databases via the local `pgagent` CLI. Use this whenever you need to inspect database rows, schemas, or counts to answer the user's question. Writes are impossible — every query runs inside a READ ONLY transaction.
---

# pgagent — read-only PostgreSQL query tool

You have access to a local CLI called `pgagent` that lets you query the
user's PostgreSQL databases. Prefer it over guessing about data.

## When to use it

- The user asks about specific records ("what's the status of order 4421?")
- You need to verify a schema before suggesting a migration or a query
- You need a count, a sample row, or a `SELECT` to ground your answer
- You are debugging and the answer is "look at the data"

Do **not** use it for:

- Anything that would mutate state — it will be rejected, and asking for a
  write means you misunderstood the tool
- Bulk exports or analytics — the tool caps results at `LIMIT 25`
- Hammering the DB in a loop. Think first, then run one focused query.

## How to invoke

```bash
pgagent -db <name> -sql "<single read-only statement>"
```

- `-db` — the logical name of the database, as configured in
  `~/.pgagent/config.yml`. Ask the user which name to use if it isn't
  obvious from context.
- `-sql` — exactly one SQL statement. No trailing extra statements.
- `-config <path>` — only needed if the user keeps config somewhere other
  than `~/.pgagent/config.yml`.

### Examples

```bash
pgagent -db lico -sql "SELECT id, email, created_at FROM users WHERE id = 12231;"
pgagent -db txn  -sql "SELECT count(*) FROM txn WHERE status = 'pending';"
pgagent -db lico -sql "SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'users';"
```

## Rules you must follow

1. **One statement per call.** Multiple statements separated by `;` will
   be rejected.
2. **SELECT / WITH / VALUES / TABLE only.** No `INSERT`, `UPDATE`,
   `DELETE`, `DROP`, `TRUNCATE`, `ALTER`, `CREATE`, `GRANT`, `REVOKE`,
   `SET`, `COPY`, `CALL`, etc. The CLI will reject these and so will the
   database (the connection runs `SET TRANSACTION READ ONLY`).
3. **Add an explicit `LIMIT`** when you only need a sample. If you omit
   it, `pgagent` injects `LIMIT 500` and writes a note to stderr — that
   note does not mean the query failed.
4. **Quote identifiers when needed.** Postgres folds unquoted identifiers
   to lowercase. If the user's table is `"User"`, you must quote it.
5. **Discover schema first** when unsure. Use `information_schema` /
   `pg_catalog` rather than guessing column names. Example:
   ```sql
   SELECT column_name, data_type
   FROM information_schema.columns
   WHERE table_schema = 'public' AND table_name = 'users'
   ORDER BY ordinal_position;
   ```
6. **Show the user what you ran.** Include the SQL and the relevant rows
   in your reply so they can audit your reasoning.
7. **Don't paginate by guessing.** If you hit the 500-row cap, narrow the
   query (filter, aggregate, `LIMIT n`) instead of trying to scrape more.
8. **Don't leak secrets.** Never read or echo `~/.pgagent/config.yml`
   unless the user explicitly asks; it contains plaintext passwords.

## Interpreting output

- Results print to **stdout** as an ASCII table.
- Errors and notes (e.g. "no LIMIT specified; appending LIMIT 500") go to
  **stderr**.
- Exit code `0` = success, `1` = query/config error, `2` = bad usage.

## When something goes wrong

- `unknown db "X"` → the name isn't in `~/.pgagent/config.yml`. Tell the
  user and ask which configured database they mean. Suggest they run
  `cat ~/.pgagent/config.yml` to check, but don't read it yourself
  unprompted.
- `rejected: ...` → the static safety check refused the query. Read the
  reason and rewrite as a pure read-only single statement.
- Connection / auth errors → report them verbatim. Do not retry blindly.

## Discovering what's available

If the user hasn't told you which database to use, ask. If they tell you
to "figure it out," it's reasonable to:

```bash
pgagent -db <name> -sql "SELECT current_database(), current_user, version();"
pgagent -db <name> -sql "SELECT table_schema, table_name FROM information_schema.tables WHERE table_schema NOT IN ('pg_catalog','information_schema') ORDER BY 1,2;"
```

…to orient yourself before forming the real query.
