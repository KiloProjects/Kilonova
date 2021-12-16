ALTER TABLE tests ADD COLUMN input_oid oid NOT NULL DEFAULT lo_create(0);
ALTER TABLE tests ADD COLUMN output_oid oid NOT NULL DEFAULT lo_create(0);

ALTER TABLE submission_tests ADD COLUMN output_oid oid NOT NULL DEFAULT lo_create(0);

BEGIN;
UPDATE tests SET input_oid = lo_import(format('/tmp/kninfo/data/tests/%s.in', id), input_oid) WHERE lo_unlink(input_oid) > -100;
UPDATE tests SET output_oid = lo_import(format('/tmp/kninfo/data/tests/%s.out', id), output_oid) WHERE lo_unlink(output_oid) > -100;
UPDATE submission_tests SET output_oid = lo_import(format('/tmp/kninfo/data/subtests/%s', id), output_oid) WHERE lo_unlink(output_oid) > -100;
COMMIT;
