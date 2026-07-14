-- Make classification_id nullable in feedback table
-- This allows feedback to be submitted without a valid classification reference

ALTER TABLE feedback 
    DROP CONSTRAINT IF EXISTS feedback_classification_id_fkey;

ALTER TABLE feedback 
    ALTER COLUMN classification_id DROP NOT NULL;

ALTER TABLE feedback 
    ADD CONSTRAINT feedback_classification_id_fkey 
    FOREIGN KEY (classification_id) 
    REFERENCES classifications(id) 
    ON DELETE SET NULL;
