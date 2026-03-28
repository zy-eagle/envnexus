ALTER TABLE model_profiles ADD COLUMN api_key VARCHAR(512) NOT NULL DEFAULT '' AFTER model_name;
