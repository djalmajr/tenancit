// Command import-legacy seeds Tenancit from the legacy CentralIT/hyperplatform
// resource-tenant tables (the front-manager DB), for one platform application.
//
// For each tenant (cluster_space_client) it reads the DATABASE resource
// (host/port/databaseName/username/password/schema/dbms), decrypts the legacy
// DES-CBC ("!@,...") values, and POSTs them to the Tenancit admin API, which
// re-encrypts secrets with AES-GCM. Idempotent: existing definitions, tenants,
// domains and resources are skipped (409s tolerated).
//
// Usage:
//
//	go run ./cmd/import-legacy \
//	  -legacy-dsn 'postgres://frontmanager:frontmanager@localhost:5433/frontmanager?sslmode=disable' \
//	  -tenancit http://localhost:8080 -admin-token tenancit_admin_dev \
//	  -app '@hyper/kanban' -limit 5 -legacy-key '<centralit-pbe-key>'
package main

import (
	"bytes"
	"context"
	"crypto/cipher"
	"crypto/des"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// legacyKey is the CentralIT/Run2Biz SecurityGenerator PBE password used to
// decrypt legacy "!@,{salt12}{base64}" values (DES-CBC, 20x MD5 derivation).
// Supplied via -legacy-key (not committed; it is a third-party legacy secret).
var legacyKey string

// postgres resource definition (field keys mirror the legacy DATABASE template).
var pgFields = []field{
	{Key: "host", Label: "Host", Required: true},
	{Key: "port", Label: "Port", Required: true},
	{Key: "databaseName", Label: "Database", Required: true},
	{Key: "username", Label: "Username", Required: true},
	{Key: "password", Label: "Password", Required: true, IsSecret: true},
	{Key: "schema", Label: "Schema"},
	{Key: "dbms", Label: "DBMS"},
}

type field struct {
	Key, Label string
	Required   bool
	IsSecret   bool
}

type tenantData struct {
	alias  string
	realm  string
	values map[string]string // field key -> decrypted value
}

func main() {
	legacyDSN := flag.String("legacy-dsn", "", "postgres DSN of the legacy front-manager DB (required)")
	tenancitURL := flag.String("tenancit", "http://localhost:8080", "Tenancit base URL")
	adminToken := flag.String("admin-token", "tenancit_admin_dev", "Tenancit admin token")
	app := flag.String("app", "@hyper/kanban", "legacy platform_application name")
	limit := flag.Int("limit", 5, "max tenants to import (0 = all)")
	legacyKeyFlag := flag.String("legacy-key", "", "CentralIT SecurityGenerator PBE key to decrypt legacy !@, values (required)")
	dryRun := flag.Bool("dry-run", false, "log what would be imported without writing")
	flag.Parse()

	if *legacyDSN == "" || *legacyKeyFlag == "" {
		log.Fatal("-legacy-dsn and -legacy-key are required")
	}
	legacyKey = *legacyKeyFlag
	ctx := context.Background()

	tenants, err := readLegacy(ctx, *legacyDSN, *app)
	if err != nil {
		log.Fatalf("read legacy: %v", err)
	}
	log.Printf("found %d tenant(s) with a DATABASE resource for app %q", len(tenants), *app)
	if *limit > 0 && len(tenants) > *limit {
		tenants = tenants[:*limit]
		log.Printf("limiting to %d", *limit)
	}

	c := &admin{base: strings.TrimRight(*tenancitURL, "/"), token: *adminToken, hc: &http.Client{Timeout: 15 * time.Second}, dry: *dryRun}

	if err := c.ensureDefinition("postgres", "PostgreSQL", "Tenant Postgres database", pgFields); err != nil {
		log.Fatalf("ensure definition: %v", err)
	}

	ok, skip := 0, 0
	for _, t := range tenants {
		slug := slugify(t.alias)
		host := t.alias // POC: resolve by alias; real hostname mapping is a later concern
		if err := c.importTenant(slug, t.alias, host, "postgres", t.values); err != nil {
			log.Printf("  tenant %s: SKIP/err: %v", slug, err)
			skip++
			continue
		}
		log.Printf("  tenant %s (host=%s) -> db=%s schema=%s user=%s", slug, host, t.values["databaseName"], t.values["schema"], t.values["username"])
		ok++
	}
	log.Printf("done: %d imported, %d skipped/failed (dryRun=%v)", ok, skip, *dryRun)
}

// readLegacy loads the DATABASE resource values per tenant (cluster_space_client)
// for the given app, decrypting legacy DES-CBC values.
func readLegacy(ctx context.Context, dsn, app string) ([]tenantData, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	defer pool.Close()

	const q = `
SELECT csc.cluster_space_client_id, csc.alias, coalesce(csc.realm,''), rfs.message_key, rtfv.value
FROM resource_tenant rt
JOIN cluster_space_client csc ON csc.cluster_space_client_id = rt.cluster_space_client_id
JOIN resource_tenant_field_value rtfv ON rtfv.resource_tenant_id = rt.resource_tenant_id
JOIN resource_field_setting rfs ON rfs.resource_field_setting_id = rtfv.resource_field_setting_id
JOIN platform_application pa ON pa.platform_application_id = rt.platform_application_id
JOIN resource_template tpl ON tpl.resource_template_id = rt.resource_template_id
WHERE pa.name = $1 AND tpl.name = 'DATABASE' AND rt.status = 'A'`

	rows, err := pool.Query(ctx, q, app)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byTenant := map[int64]*tenantData{}
	var order []int64
	for rows.Next() {
		var id int64
		var alias, realm, label string
		var value *string // value can be NULL in the legacy table
		if err := rows.Scan(&id, &alias, &realm, &label, &value); err != nil {
			return nil, err
		}
		t, ok := byTenant[id]
		if !ok {
			t = &tenantData{alias: alias, realm: realm, values: map[string]string{}}
			byTenant[id] = t
			order = append(order, id)
		}
		raw := ""
		if value != nil {
			raw = *value
		}
		dec, err := legacyDecrypt(raw)
		if err != nil {
			return nil, fmt.Errorf("decrypt %s for tenant %s: %w", label, alias, err)
		}
		t.values[label] = dec
	}
	out := make([]tenantData, 0, len(order))
	for _, id := range order {
		out = append(out, *byTenant[id])
	}
	return out, rows.Err()
}

// legacyDecrypt reverses the CentralIT SecurityGenerator (DES-CBC + PBE).
func legacyDecrypt(v string) (string, error) {
	v = strings.TrimSpace(v)
	if !strings.HasPrefix(v, "!@,") {
		return v, nil // plaintext
	}
	data := v[len("!@,"):]
	if len(data) < 12 {
		return "", fmt.Errorf("ciphertext too short")
	}
	saltStr, cipherB64 := data[:12], data[12:]
	salt, err := base64.StdEncoding.DecodeString(saltStr)
	if err != nil {
		return "", fmt.Errorf("salt b64: %w", err)
	}
	derived := append([]byte(legacyKey), salt...)
	for i := 0; i < 20; i++ {
		sum := md5.Sum(derived)
		derived = sum[:]
	}
	key, iv := derived[:8], derived[8:16]
	ct, err := base64.StdEncoding.DecodeString(cipherB64)
	if err != nil {
		return "", fmt.Errorf("cipher b64: %w", err)
	}
	block, err := des.NewCipher(key)
	if err != nil {
		return "", err
	}
	if len(ct) == 0 || len(ct)%des.BlockSize != 0 {
		return "", fmt.Errorf("bad ciphertext length")
	}
	pt := make([]byte, len(ct))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(pt, ct)
	// strip PKCS#5/7 padding
	n := int(pt[len(pt)-1])
	if n <= 0 || n > des.BlockSize || n > len(pt) {
		return "", fmt.Errorf("bad padding")
	}
	return string(pt[:len(pt)-n]), nil
}

var nonSlug = regexp.MustCompile(`[^a-z0-9-]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonSlug.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// --- Tenancit admin client ---

type admin struct {
	base, token string
	hc          *http.Client
	dry         bool
}

func (a *admin) do(method, path string, body any) (int, []byte, error) {
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, a.base+path, rdr)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+a.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := a.hc.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, rb, nil
}

func (a *admin) ensureDefinition(key, name, desc string, fields []field) error {
	if a.dry {
		log.Printf("[dry] ensure definition %s + %d fields", key, len(fields))
		return nil
	}
	st, rb, err := a.do("POST", "/v1/admin/resource-definitions", map[string]string{"key": key, "name": name, "description": desc})
	if err != nil {
		return err
	}
	if st != http.StatusCreated && st != http.StatusConflict {
		return fmt.Errorf("create definition: %d %s", st, rb)
	}
	defID, err := a.definitionID(key)
	if err != nil {
		return err
	}
	for i, f := range fields {
		fst, frb, err := a.do("POST", "/v1/admin/resource-definitions/"+defID+"/fields", map[string]any{
			"key": f.Key, "label": f.Label, "dataType": "string", "required": f.Required, "isSecret": f.IsSecret, "sortOrder": i,
		})
		if err != nil {
			return err
		}
		if fst != http.StatusCreated && fst != http.StatusConflict && fst != http.StatusOK {
			return fmt.Errorf("add field %s: %d %s", f.Key, fst, frb)
		}
	}
	log.Printf("definition %q ready (%d fields)", key, len(fields))
	return nil
}

func (a *admin) definitionID(key string) (string, error) {
	st, rb, err := a.do("GET", "/v1/admin/resource-definitions", nil)
	if err != nil {
		return "", err
	}
	if st != http.StatusOK {
		return "", fmt.Errorf("list definitions: %d %s", st, rb)
	}
	var defs []struct {
		ID, Key string
	}
	if err := json.Unmarshal(rb, &defs); err != nil {
		return "", err
	}
	for _, d := range defs {
		if d.Key == key {
			return d.ID, nil
		}
	}
	return "", fmt.Errorf("definition %q not found after create", key)
}

func (a *admin) importTenant(slug, name, hostname, defKey string, values map[string]string) error {
	if a.dry {
		log.Printf("[dry] tenant %s host=%s values=%v", slug, hostname, redact(values))
		return nil
	}
	// create tenant (or reuse on conflict)
	id, err := a.ensureTenant(slug, name)
	if err != nil {
		return err
	}
	// domain (idempotent)
	dst, drb, err := a.do("POST", "/v1/admin/tenants/"+id+"/domains", map[string]string{"hostname": hostname})
	if err != nil {
		return err
	}
	if dst != http.StatusCreated && dst != http.StatusConflict {
		return fmt.Errorf("add domain: %d %s", dst, drb)
	}
	// resource
	rst, rrb, err := a.do("POST", "/v1/admin/tenants/"+id+"/resources", map[string]any{"definitionKey": defKey, "values": values})
	if err != nil {
		return err
	}
	if rst != http.StatusCreated && rst != http.StatusConflict {
		return fmt.Errorf("create resource: %d %s", rst, rrb)
	}
	return nil
}

func (a *admin) ensureTenant(slug, name string) (string, error) {
	st, rb, err := a.do("POST", "/v1/admin/tenants", map[string]string{"slug": slug, "name": name})
	if err != nil {
		return "", err
	}
	if st == http.StatusCreated {
		var t struct{ ID string }
		if err := json.Unmarshal(rb, &t); err != nil {
			return "", err
		}
		return t.ID, nil
	}
	if st != http.StatusConflict {
		return "", fmt.Errorf("create tenant: %d %s", st, rb)
	}
	// conflict: find existing by slug
	lst, lrb, err := a.do("GET", "/v1/admin/tenants", nil)
	if err != nil {
		return "", err
	}
	var ts []struct{ ID, Slug string }
	if err := json.Unmarshal(lrb, &ts); err != nil {
		return "", fmt.Errorf("list tenants: %d %w", lst, err)
	}
	for _, t := range ts {
		if t.Slug == slug {
			return t.ID, nil
		}
	}
	return "", fmt.Errorf("tenant %q not found after conflict", slug)
}

func redact(v map[string]string) map[string]string {
	out := map[string]string{}
	for k, val := range v {
		if k == "password" {
			out[k] = "***"
		} else {
			out[k] = val
		}
	}
	return out
}
