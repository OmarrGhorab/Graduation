DO $$ 
BEGIN 
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'billing_type') THEN
        CREATE TYPE billing_type AS ENUM ('ONE_TIME', 'MONTHLY');
    END IF;
END $$;

ALTER TABLE courses ADD COLUMN billing_type billing_type NOT NULL DEFAULT 'ONE_TIME';
