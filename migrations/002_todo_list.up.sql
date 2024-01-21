--
-- Todo lists
--

CREATE TABLE todo_lists (
  todo_list_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE todo_lists_history (
  -- copy these fields, always keep them at the top
  history_id UUID PRIMARY KEY,
  systime TSTZRANGE NOT NULL CHECK (NOT ISEMPTY(systime)),

  -- table fields, in the exact same order as in todo_list
  todo_list_id UUID NOT NULL,
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);


ALTER TABLE todo_lists_history
  ADD CONSTRAINT todo_lists_history_overlapping_excl
  EXCLUDE USING GIST (todo_list_id WITH =, systime WITH &&);

CREATE TRIGGER todo_lists_history_insert_delete_trigger
AFTER INSERT OR DELETE ON todo_lists
    FOR EACH ROW
    EXECUTE PROCEDURE copy_inserts_and_deletes_into_history('todo_lists_history', 'todo_list_id');

CREATE TRIGGER todo_lists_history_update_trigger
AFTER UPDATE ON todo_lists
    FOR EACH ROW
    WHEN (OLD.* IS DISTINCT FROM NEW.*) -- to avoid updates on "noop calls"
    EXECUTE PROCEDURE copy_updates_into_history('todo_lists_history', 'todo_list_id');

--
-- Todos
--

CREATE TABLE todos (
  todo_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  todo_list_id UUID NOT NULL REFERENCES todo_lists(todo_list_id) ON DELETE CASCADE,
  description TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX todos_todo_list_id_idx
  ON todos (todo_list_id, todo_id);

CREATE TABLE todos_history (
  -- copy these fields, always keep them at the top
  history_id UUID PRIMARY KEY,
  systime TSTZRANGE NOT NULL CHECK (NOT ISEMPTY(systime)),

  -- table fields, in the exact same order as in todos
  todo_id UUID NOT NULL,
  todo_list_id UUID NOT NULL,
  description TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  completed BOOLEAN NOT NULL
);

CREATE INDEX todos_history_todo_list_id
  ON todos_history USING GIST (todo_list_id, systime);

ALTER TABLE todos_history
  ADD CONSTRAINT todos_history_overlapping_excl
  EXCLUDE USING GIST (todo_id WITH =, systime WITH &&);

CREATE TRIGGER todos_history_insert_delete_trigger
AFTER INSERT OR DELETE ON todos
    FOR EACH ROW
    EXECUTE PROCEDURE copy_inserts_and_deletes_into_history('todos_history', 'todo_id');

CREATE TRIGGER todos_history_update_trigger
AFTER UPDATE ON todos
    FOR EACH ROW
    WHEN (OLD.* IS DISTINCT FROM NEW.*) -- to avoid updates on "noop calls"
    EXECUTE PROCEDURE copy_updates_into_history('todos_history', 'todo_id');
