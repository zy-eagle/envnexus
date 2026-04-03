-- Platform super admins see all tenants and may approve/deny any command task (any tenant).
-- Normal tenant operators only see their home tenant and only approve tasks they are assigned to (user or role).

ALTER TABLE users
    ADD COLUMN platform_super_admin TINYINT(1) NOT NULL DEFAULT 0
        AFTER status;

UPDATE users SET platform_super_admin = 1 WHERE LOWER(email) = 'admin@gmail.com';

INSERT INTO users (id, tenant_id, email, display_name, password_hash, status, platform_super_admin, created_at, updated_at)
SELECT '01J9PLATFORMSUPERADMIN0001', '01JTENANT0001DEFAULTADMIN0', 'admin@gmail.com',
       'Platform Super Admin', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'active', 1, NOW(3), NOW(3)
FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM users WHERE LOWER(email) = 'admin@gmail.com');
