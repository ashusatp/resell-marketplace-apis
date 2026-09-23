ALTER TABLE listings ADD COLUMN created_by UUID;
ALTER TABLE listings ADD CONSTRAINT fk_listings_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_listings_created_by ON listings(created_by);