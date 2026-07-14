-- Revert classification_id to required in feedback table

ALTER TABLE feedback 
    DROP CONSTRAINT IF EXISTS feedback_classification_id_fkey;

-- Delete any feedback with null classification_id before making it required
DELETE FROM feedback WHERE classification_id IS NULL;

ALTER TABLE feedback 
    ALTER COLUMN classification_id SET NOT NULL;

ALTER TABLE feedback 
    ADD CONSTRAINT feedback_classification_id_fkey 
    FOREIGN KEY (classification_id) 
    REFERENCES classifications(id) 
    ON DELETE CASCADE;
