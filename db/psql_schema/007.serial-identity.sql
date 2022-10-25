-- credit: https://www.2ndquadrant.com/en/blog/postgresql-10-identity-columns/
CREATE OR REPLACE FUNCTION upgrade_serial_to_identity(tbl regclass, col name)
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
  colnum smallint;
  seqid oid;
  count int;
BEGIN
  -- find column number
  SELECT attnum INTO colnum FROM pg_attribute WHERE attrelid = tbl AND attname = col;
  IF NOT FOUND THEN
    RAISE EXCEPTION 'column does not exist';
  END IF;

  -- find sequence
  SELECT INTO seqid objid
    FROM pg_depend
    WHERE (refclassid, refobjid, refobjsubid) = ('pg_class'::regclass, tbl, colnum)
      AND classid = 'pg_class'::regclass AND objsubid = 0
      AND deptype = 'a';

  GET DIAGNOSTICS count = ROW_COUNT;
  IF count < 1 THEN
    RAISE EXCEPTION 'no linked sequence found';
  ELSIF count > 1 THEN
    RAISE EXCEPTION 'more than one linked sequence found';
  END IF;  

  -- drop the default
  EXECUTE 'ALTER TABLE ' || tbl || ' ALTER COLUMN ' || quote_ident(col) || ' DROP DEFAULT';

  -- change the dependency between column and sequence to internal
  UPDATE pg_depend
    SET deptype = 'i'
    WHERE (classid, objid, objsubid) = ('pg_class'::regclass, seqid, 0)
      AND deptype = 'a';

  -- mark the column as identity column
  UPDATE pg_attribute
    SET attidentity = 'd'
    WHERE attrelid = tbl
      AND attname = col;
END;
$$;

SELECT upgrade_serial_to_identity('users', 'id');
SELECT upgrade_serial_to_identity('problems', 'id');
SELECT upgrade_serial_to_identity('tests', 'id');
SELECT upgrade_serial_to_identity('submissions', 'id');
SELECT upgrade_serial_to_identity('submission_tests', 'id');
SELECT upgrade_serial_to_identity('problem_lists', 'id');
SELECT upgrade_serial_to_identity('subtasks', 'id');
SELECT upgrade_serial_to_identity('attachments', 'id');

