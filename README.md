# Time Travelling Todo Lists in Postgres

An implementation of system-versioned tables in Postgres using only triggers,
along with a TODO list app with time travelling magic on top.

## How to Run

This implementation uses Docker for spinning up a new Postgres database, and Go
for the app itself. Here's a oneliner to (re)create the database and to start
the app:


```shell
$ ./setup-db.sh && go build && ./time-travelling-todo-lists-in-postgres
```

If you want to hack on the project, I recommend using
[air](https://github.com/cosmtrek/air) to get fast feedback:

```shell
$ ./setup-db.sh
$ air
```

If you want to inspect the contents of the database itself, you can access it
like so:

```sh
$ PGPASSWORD=mySecretPassword psql -h localhost -p 10840 -U postgres postgres
```

## Why System-Versioned/Temporal Tables

I made the blog post ["Implementing System-Versioned Tables in
Postgres"](https://hypirion.com/musings/implementing-system-versioned-tables-in-postgres).
It also goes further into the details on how the.

## How to use it yourself

I recommend reading the blog post to grok how the triggers work under the
covers. Here I'll explain how to use it and the things you need to be aware of
if you want to use this technique.

First off, make a migration for the history triggers just like in
`migrations/001_history_triggers.up.sql`. Then, take the tables you want
system-versioned and make a copy named `xxx_history`. The history tables must
follow this shape:

```sql
CREATE TABLE mytable_history (
  -- copy these columns, always keep them at the top
  history_id UUID PRIMARY KEY,
  systime TSTZRANGE NOT NULL CHECK (NOT ISEMPTY(systime)),

  -- table columns, in the exact same order as in mytable
  mytable_id UUID NOT NULL,
  more_columns TEXT NOT NULL
);
```

Be sure that the order of your columns are in the same order as in the original
table, otherwise will end up with broken triggers that either break CUD
operations on the original table, or even worse, silently insert corrupted data
into the history table.

If you are unsure of the ordering, you can use `psql` and issue the command `\d
mytable` to see which order they are stored in.

Future changes must always happen in pairs. For example:

```sql
ALTER TABLE mytable
  ADD COLUMN more_columns TEXT NOT NULL DEFAULT 'default-value';

ALTER TABLE mytable_history
  ADD COLUMN more_columns TEXT NOT NULL DEFAULT 'default-value';
```

The history tables won't be able to have any reasonable foreign keys, though as
long as they contain the exact same shape as the snapshot table, that's not a
problem. However, if you manipulate the history tables yourself, you may end up
with dangling references (i.e: don't do that..).

It is possible to include foreign keys in the history table to ensure you adhere
to e.g. GDPR. I may show how to do that at some point in the future.

---

Next you have to add the primary key and the triggers. The primary key is a GiST
index, where the original primary key is compared with `=` (add them in sequence
if the key is compound), and the system time is at the end, compared with `&&`.

```sql
ALTER TABLE mytable_history
  ADD CONSTRAINT mytable_history_overlapping_excl
  EXCLUDE USING GIST (mytable_id WITH =, systime WITH &&);
```

Note that the GiST index will ensure that there's only one row matching the
original primary key at any given time instant. **This means that concurrent
transactions changing the same row is likely to cause one of them to fail.** If
this is an issue for your use case, you have three options:

1. Don't use system-versioned tables, but rather an event table or something
   similar
2. Keep the GiST index, but modify/remove it as a constraint
3. Apply retry logic for transactions prone to the issue

In my eyes, I'd use system-versioned tables for things users trigger, or things
that doesn't change so fast that the GiST index causes a problem in practice.


Creating the triggers is done as such:

```sql
CREATE TRIGGER mytable_history_insert_delete_trigger
AFTER INSERT OR DELETE ON mytable
    FOR EACH ROW
    EXECUTE PROCEDURE copy_inserts_and_deletes_into_history('mytable_history', 'mytable_id');

CREATE TRIGGER mytable_history_update_trigger
AFTER UPDATE ON mytable
    FOR EACH ROW
    WHEN (OLD.* IS DISTINCT FROM NEW.*) -- to avoid updates on "noop calls"
    EXECUTE PROCEDURE copy_updates_into_history('mytable_history', 'mytable_id');
```

If you want to, you can remove the `WHEN (OLD.* IS DISTINCT FROM NEW.*)` call,
though it likely doesn't make much sense to do so.


## History Queries

There are a couple of history queries over at `todos_history.go`, and they are
for the most part straightforward. If you want rows that were valid at some
point in time, use:

```sql
mytable_history.systime @> CAST(:as_of AS timestamptz)
```

If you want to join over multiple history tables, be sure to include the systime
clause for all of them, e.g.:

```sql
SELECT m1.col1, m2.col2
FROM mytable1_history m1
JOIN mytable2_history m2 ON m1.some_id = m2.some_id
WHERE m1.systime @> CAST(:as_of AS timestamptz)
  AND m2.systime @> CAST(:as_of AS timestamptz)
  AND m1.some_other_id = :myid;
```

If you don't, you'll probably get more rows than you wanted.

It's generally a bad idea to join history tables with non-history tables. The
exception is if the table is an append-only table or an event log of some kind.

## License

I've waived my ownership to this by applying a CC0 license to this repo. Do
whatever you want with the code that resides here, although I don't mind a
referral back to this repository or to my original blog post.
