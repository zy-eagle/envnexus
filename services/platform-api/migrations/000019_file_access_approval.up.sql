-- Add approval policy fields to file_access_requests for download approval flow
ALTER TABLE file_access_requests
    ADD COLUMN approver_user_id CHAR(26) NULL AFTER approved_by,
    ADD COLUMN approver_role_id CHAR(26) NULL AFTER approver_user_id,
    ADD COLUMN policy_snapshot_id CHAR(26) NULL AFTER approver_role_id;
