-- +goose Up
-- resource:write — consumer scope to update values of NON-SECRET fields on
-- existing resources (tenant slug + alias addressing). Secret fields are
-- always rejected at the API layer; secret management stays in the console.
ALTER TABLE api_client_scopes DROP CONSTRAINT api_client_scopes_scope_check;
ALTER TABLE api_client_scopes ADD CONSTRAINT api_client_scopes_scope_check
    CHECK (scope IN ('tenant:identify', 'resource:resolve', 'events:read', 'tenant:list', 'resource:write'));
ALTER TABLE api_client_usage_daily DROP CONSTRAINT api_client_usage_daily_operation_check;
ALTER TABLE api_client_usage_daily ADD CONSTRAINT api_client_usage_daily_operation_check
    CHECK (operation IN ('identify', 'resolve', 'events', 'tenants', 'write'));

-- +goose Down
DELETE FROM api_client_usage_daily WHERE operation = 'write';
ALTER TABLE api_client_usage_daily DROP CONSTRAINT api_client_usage_daily_operation_check;
ALTER TABLE api_client_usage_daily ADD CONSTRAINT api_client_usage_daily_operation_check
    CHECK (operation IN ('identify', 'resolve', 'events', 'tenants'));
DELETE FROM api_client_scopes WHERE scope = 'resource:write';
ALTER TABLE api_client_scopes DROP CONSTRAINT api_client_scopes_scope_check;
ALTER TABLE api_client_scopes ADD CONSTRAINT api_client_scopes_scope_check
    CHECK (scope IN ('tenant:identify', 'resource:resolve', 'events:read', 'tenant:list'));
