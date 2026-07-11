-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM api_clients
        WHERE token_preview IS NULL OR rpm_limit IS NULL OR expires_at IS NULL
    ) THEN
        RAISE EXCEPTION 'api client governance contract blocked: legacy_unbounded clients remain';
    END IF;
    IF EXISTS (
        SELECT 1 FROM api_clients c
        WHERE NOT EXISTS (SELECT 1 FROM api_client_scopes s WHERE s.api_client_id = c.id)
    ) THEN
        RAISE EXCEPTION 'api client governance contract blocked: clients without scopes remain';
    END IF;
END;
$$;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS api_clients_grant_legacy_scopes ON api_clients;
DROP FUNCTION IF EXISTS grant_legacy_api_client_scopes();

ALTER TABLE api_clients
    ALTER COLUMN token_preview SET NOT NULL,
    ALTER COLUMN rpm_limit SET NOT NULL,
    ALTER COLUMN expires_at SET NOT NULL;

ALTER TABLE api_clients VALIDATE CONSTRAINT api_clients_rpm_limit_positive;
ALTER TABLE api_clients VALIDATE CONSTRAINT api_clients_expiration_after_creation;

-- +goose StatementBegin
CREATE FUNCTION reject_api_client_reactivation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.status = 'revoked' AND NEW.status <> 'revoked' THEN
        RAISE EXCEPTION 'revoked API clients cannot be reactivated';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER api_clients_revocation_terminal
BEFORE UPDATE OF status ON api_clients
FOR EACH ROW EXECUTE FUNCTION reject_api_client_reactivation();

-- +goose Down
DROP TRIGGER IF EXISTS api_clients_revocation_terminal ON api_clients;
DROP FUNCTION IF EXISTS reject_api_client_reactivation();
ALTER TABLE api_clients
    ALTER COLUMN expires_at DROP NOT NULL,
    ALTER COLUMN rpm_limit DROP NOT NULL,
    ALTER COLUMN token_preview DROP NOT NULL;

-- +goose StatementBegin
CREATE FUNCTION grant_legacy_api_client_scopes() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO api_client_scopes (api_client_id, scope)
    VALUES (NEW.id, 'tenant:identify'), (NEW.id, 'resource:resolve');
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER api_clients_grant_legacy_scopes
AFTER INSERT ON api_clients
FOR EACH ROW EXECUTE FUNCTION grant_legacy_api_client_scopes();
