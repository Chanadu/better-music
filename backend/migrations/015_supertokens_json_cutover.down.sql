DO $$ BEGIN
    RAISE EXCEPTION 'The SuperTokens cutover requires restoring a pre-cutover backup to roll back';
END $$;
