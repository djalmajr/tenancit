-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE tenants (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug       text NOT NULL UNIQUE,
    name       text NOT NULL,
    status     text NOT NULL DEFAULT 'active',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE tenant_domains (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    hostname   text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_tenant_domains_tenant ON tenant_domains(tenant_id);

CREATE TABLE resource_definitions (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    key         text NOT NULL UNIQUE,
    name        text NOT NULL,
    description text NOT NULL DEFAULT '',
    icon        text NOT NULL DEFAULT '',
    status      text NOT NULL DEFAULT 'active',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE resource_fields (
    id                     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_definition_id uuid NOT NULL REFERENCES resource_definitions(id) ON DELETE CASCADE,
    key                    text NOT NULL,
    label                  text NOT NULL DEFAULT '',
    hint                   text NOT NULL DEFAULT '',
    data_type              text NOT NULL DEFAULT 'string',
    required               boolean NOT NULL DEFAULT false,
    is_secret              boolean NOT NULL DEFAULT false,
    sort_order             integer NOT NULL DEFAULT 0,
    UNIQUE (resource_definition_id, key)
);

CREATE TABLE tenant_resources (
    id                     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    resource_definition_id uuid NOT NULL REFERENCES resource_definitions(id),
    status                 text NOT NULL DEFAULT 'active',
    created_at             timestamptz NOT NULL DEFAULT now(),
    updated_at             timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uq_tenant_resource_active
    ON tenant_resources(tenant_id, resource_definition_id)
    WHERE status = 'active';

CREATE TABLE tenant_resource_values (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_resource_id uuid NOT NULL REFERENCES tenant_resources(id) ON DELETE CASCADE,
    resource_field_id  uuid NOT NULL REFERENCES resource_fields(id),
    value_plain        text,
    value_cipher       bytea,
    nonce              bytea,
    key_version        integer,
    UNIQUE (tenant_resource_id, resource_field_id)
);

CREATE TABLE api_clients (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL,
    key_hash   text NOT NULL UNIQUE,
    status     text NOT NULL DEFAULT 'active',
    created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS api_clients;
DROP TABLE IF EXISTS tenant_resource_values;
DROP TABLE IF EXISTS tenant_resources;
DROP TABLE IF EXISTS resource_fields;
DROP TABLE IF EXISTS resource_definitions;
DROP TABLE IF EXISTS tenant_domains;
DROP TABLE IF EXISTS tenants;
