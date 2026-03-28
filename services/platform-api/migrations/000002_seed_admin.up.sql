-- Seed default tenant and admin user
-- Password: 'admin123' hashed with bcrypt
INSERT INTO tenants (id, name, slug, status, created_at, updated_at)
VALUES ('01JTENANT0001DEFAULTADMIN0', 'Default Tenant', 'default', 'active', NOW(3), NOW(3));

INSERT INTO users (id, tenant_id, email, display_name, password_hash, status, created_at, updated_at)
VALUES ('01JUSER00001DEFAULTADMIN00', '01JTENANT0001DEFAULTADMIN0', 'admin@envnexus.local',
        'Admin', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'active', NOW(3), NOW(3));

INSERT INTO roles (id, tenant_id, name, permissions_json, status, created_at, updated_at)
VALUES ('01JROLE00001TENANTADMIN000', '01JTENANT0001DEFAULTADMIN0', 'tenant_admin',
        '{"manage_tenants":true,"manage_profiles":true,"manage_devices":true,"manage_packages":true,"view_audit":true,"manage_users":true}',
        'active', NOW(3), NOW(3));
