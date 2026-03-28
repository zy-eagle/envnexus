-- Fix unique indexes on profile tables to account for soft-deleted records.
-- Uses IFNULL to map NULL deleted_at (active records) to a sentinel value,
-- ensuring true uniqueness among non-deleted records while allowing
-- re-creation of a name after the previous record is soft-deleted.

ALTER TABLE agent_profiles DROP INDEX uk_agent_profiles_tenant_name;
CREATE UNIQUE INDEX uk_agent_profiles_tenant_name
    ON agent_profiles (tenant_id, name, (IFNULL(deleted_at, '1970-01-01 00:00:00')));

ALTER TABLE model_profiles DROP INDEX uk_model_profiles_tenant_name;
CREATE UNIQUE INDEX uk_model_profiles_tenant_name
    ON model_profiles (tenant_id, name, (IFNULL(deleted_at, '1970-01-01 00:00:00')));

ALTER TABLE policy_profiles DROP INDEX uk_policy_profiles_tenant_name;
CREATE UNIQUE INDEX uk_policy_profiles_tenant_name
    ON policy_profiles (tenant_id, name, (IFNULL(deleted_at, '1970-01-01 00:00:00')));
