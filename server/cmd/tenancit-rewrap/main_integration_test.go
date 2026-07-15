package main

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	appcrypto "github.com/djalmajr/tenancit/server/internal/crypto"
	"github.com/djalmajr/tenancit/server/internal/testsupport"
	"github.com/google/uuid"
)

func TestExecuteRunsDryRunAndConfirmedCampaignWithoutKeyArguments(t *testing.T) {
	pool := testsupport.NewDB(t)
	key1 := bytes.Repeat([]byte{0x31}, 32)
	key2 := bytes.Repeat([]byte{0x32}, 32)
	oldCryptor, err := appcrypto.New(map[int][]byte{1: key1}, 1)
	if err != nil {
		t.Fatal(err)
	}
	var resourceID, fieldID uuid.UUID
	if err := pool.QueryRow(t.Context(), `WITH tenant AS (INSERT INTO tenants(slug,name) VALUES('cli-rewrap','CLI') RETURNING id),
		definition AS (INSERT INTO resource_definitions(key,name) VALUES('cli-rewrap','CLI') RETURNING id),
		resource AS (INSERT INTO tenant_resources(tenant_id,resource_definition_id,display_name) SELECT tenant.id,definition.id,'CLI resource' FROM tenant,definition RETURNING id,resource_definition_id),
		field AS (INSERT INTO resource_fields(resource_definition_id,key,label,is_secret) SELECT resource_definition_id,'secret','Secret',true FROM resource RETURNING id)
		SELECT resource.id,field.id FROM resource,field`).Scan(&resourceID, &fieldID); err != nil {
		t.Fatal(err)
	}
	encrypted, err := oldCryptor.Encrypt("cli-secret-canary")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `INSERT INTO tenant_resource_values(tenant_resource_id,resource_field_id,value_cipher,nonce,key_version) VALUES($1,$2,$3,$4,$5)`, resourceID, fieldID, encrypted.Cipher, encrypted.Nonce, 1); err != nil {
		t.Fatal(err)
	}
	var resourceUpdatedBefore time.Time
	if err := pool.QueryRow(t.Context(), `SELECT updated_at FROM tenant_resources WHERE id=$1`, resourceID).Scan(&resourceUpdatedBefore); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"backup", "restore"} {
		if _, err := pool.Exec(t.Context(), `INSERT INTO operational_reports(kind,source,status,occurred_at,fresh_until,idempotency_key,payload_hash,credential_version) VALUES($1,'cli-test','healthy',clock_timestamp(),clock_timestamp()+interval '1 hour',$1||gen_random_uuid(),decode('00','hex'),'test')`, kind); err != nil {
			t.Fatal(err)
		}
	}

	reports := 0
	reportServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		reports++
		if request.Header.Get("Authorization") != "Bearer operations-report-token-long-enough" {
			t.Error("missing reporter credential")
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer reportServer.Close()
	t.Setenv("TENANCIT_AES_KEY", base64.StdEncoding.EncodeToString(key2))
	t.Setenv("TENANCIT_AES_KEY_VERSION", "2")
	t.Setenv("TENANCIT_AES_KEY_V1", base64.StdEncoding.EncodeToString(key1))
	t.Setenv("TENANCIT_REWRAP_DATABASE_URL", pool.Config().ConnString())
	t.Setenv("TENANCIT_OPERATIONS_BASE_URL", reportServer.URL)
	t.Setenv("TENANCIT_OPERATIONS_REPORT_TOKEN", "operations-report-token-long-enough")
	t.Setenv("TENANCIT_REWRAP_SOURCE", "cli-test")
	t.Setenv("TENANCIT_DEV_MODE", "true")

	dryOutput := &bytes.Buffer{}
	if err := execute(t.Context(), os.Getenv, []string{"--dry-run", "--target-version", "2", "--batch-size", "1", "--max-duration", "1m"}, dryOutput); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(dryOutput.Bytes(), []byte(`"rows_remaining":1`)) || reports != 0 {
		t.Fatalf("dry output=%s reports=%d", dryOutput.String(), reports)
	}

	writeOutput := &bytes.Buffer{}
	if err := execute(t.Context(), os.Getenv, []string{"--confirm-write", "--target-version", "2", "--batch-size", "1", "--max-duration", "1m", "--job-id", uuid.NewString()}, writeOutput); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(writeOutput.Bytes(), []byte(`"rows_rewrapped":1`)) || reports != 1 {
		t.Fatalf("write output=%s reports=%d", writeOutput.String(), reports)
	}
	var version int
	var updatedAt time.Time
	if err := pool.QueryRow(t.Context(), `SELECT v.key_version,r.updated_at FROM tenant_resource_values v JOIN tenant_resources r ON r.id=v.tenant_resource_id`).Scan(&version, &updatedAt); err != nil {
		t.Fatal(err)
	}
	if version != 2 {
		t.Fatalf("version=%d", version)
	}
	if !resourceUpdatedBefore.Equal(updatedAt) {
		t.Fatal("CLI rewrap changed functional resource timestamp")
	}
}
