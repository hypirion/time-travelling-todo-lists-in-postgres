CREATE EXTENSION btree_gist;

CREATE FUNCTION copy_inserts_and_deletes_into_history() RETURNS TRIGGER AS $$
DECLARE
  history_table TEXT := quote_ident(tg_argv[0]);
  id_field TEXT := quote_ident(tg_argv[1]);
BEGIN
  IF (TG_OP = 'INSERT') THEN
    EXECUTE 'INSERT INTO ' || history_table ||
      ' SELECT gen_random_uuid(), tstzrange(NOW(), null), $1.*'
      USING NEW;
    RETURN NEW;
  ELSIF (TG_OP = 'DELETE') THEN
    -- close current row
    -- note: updates and then deletes for same id
    -- in same tx will fail
    EXECUTE 'UPDATE ' || history_table ||
      ' SET systime = tstzrange(lower(systime), NOW())' ||
      ' WHERE ' || id_field || ' = $1.' || id_field ||
      ' AND systime @> NOW()' USING OLD;
    RETURN OLD;
  END IF;
  RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE FUNCTION copy_updates_into_history() RETURNS TRIGGER AS $$
DECLARE
  history_table TEXT := quote_ident(tg_argv[0]);
  id_field TEXT := quote_ident(tg_argv[1]);
BEGIN
  -- ignore changes inside the same tx
  EXECUTE 'DELETE FROM ' || history_table ||
    ' WHERE ' || id_field || ' = $1.' || id_field ||
    ' AND lower(systime) = NOW()' ||
    ' AND upper_inf(systime)' USING NEW;
  -- close current row
  -- (if any, may be deleted by previous line)
  EXECUTE 'UPDATE ' || history_table ||
    ' SET systime = tstzrange(lower(systime), NOW())'
    ' WHERE ' || id_field || ' = $1.' || id_field ||
    ' AND systime @> NOW()' USING NEW;
  -- insert new row
  EXECUTE 'INSERT INTO ' || history_table ||
    ' SELECT gen_random_uuid(), tstzrange(NOW(), null), $1.*'
    USING NEW;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
