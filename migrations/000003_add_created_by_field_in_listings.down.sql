DROP INDEX IF EXISTS idx_listings_created_by;
ALTER TABLE listings DROP CONSTRAINT IF EXISTS fk_listings_created_by;
ALTER TABLE listings DROP COLUMN IF EXISTS created_by;
