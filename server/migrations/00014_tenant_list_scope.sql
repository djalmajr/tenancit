-- +goose Up
-- tenant:list — consumer scope to enumerate tenant identities (slug/name/status
-- only). Generic directory access for control-plane bridges; never resources
-- or secrets.
ALTER TABLE api_client_scopes DROP CONSTRAINT api_client_scopes_scope_check;
ALTER TABLE api_client_scopes ADD CONSTRAINT api_client_scopes_scope_check
    CHECK (scope IN ('tenant:identify', 'resource:resolve', 'events:read', 'tenant:list'));
ALTER TABLE api_client_usage_daily DROP CONSTRAINT api_client_usage_daily_operation_check;
ALTER TABLE api_client_usage_daily ADD CONSTRAINT api_client_usage_daily_operation_check
    CHECK (operation IN ('identify', 'resolve', 'events', 'tenants'));

-- +goose Down
DELETE FROM api_client_usage_daily WHERE operation = 'tenants';
ALTER TABLE api_client_usage_daily DROP CONSTRAINT api_client_usage_daily_operation_check;
ALTER TABLE api_client_usage_daily ADD CONSTRAINT api_client_usage_daily_operation_check
    CHECK (operation IN ('identify', 'resolve', 'events'));
DELETE FROM api_client_scopes WHERE scope = 'tenant:list';
ALTER TABLE api_client_scopes DROP CONSTRAINT api_client_scopes_scope_check;
ALTER TABLE api_client_scopes ADD CONSTRAINT api_client_scopes_scope_check
    CHECK (scope IN ('tenant:identify', 'resource:resolve', 'events:read'));
