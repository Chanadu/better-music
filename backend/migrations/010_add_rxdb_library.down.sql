DO $$ BEGIN RAISE EXCEPTION 'RxDB cutover requires a reconciled restore; automatic down migration would lose new edits'; END $$;
