package events

import (
	"strings"
	"testing"
)

func TestFromAuditPublishesOnlyVersionedReferencePayloads(t *testing.T) {
	draft, publish, err := FromAudit("resource.provisioned", "resource", "resource-1")
	if err != nil || !publish {
		t.Fatalf("FromAudit publish=%v err=%v", publish, err)
	}
	if draft.Type != "tenancit.tenant_resource.provisioned" || draft.Version != 1 {
		t.Fatalf("draft=%+v", draft)
	}
	payload := string(draft.Payload)
	if !strings.Contains(payload, `"id":"resource-1"`) || strings.Contains(payload, "secret") || strings.Contains(payload, "token") {
		t.Fatalf("unsafe event payload=%s", payload)
	}
}

func TestFromAuditIgnoresNonDomainAndRevealEvents(t *testing.T) {
	for _, action := range []string{"secret.revealed", "admin.request_failed", "settings.updated"} {
		if _, publish, err := FromAudit(action, "target", "id"); err != nil || publish {
			t.Fatalf("action %q publish=%v err=%v", action, publish, err)
		}
	}
}

func TestFromAuditPublishesDomainRename(t *testing.T) {
	draft, publish, err := FromAudit("domain.updated", "domain", "domain-1")
	if err != nil || !publish {
		t.Fatalf("FromAudit publish=%v err=%v", publish, err)
	}
	if draft.Type != "tenancit.tenant_domain.updated" || draft.AggregateID != "domain-1" {
		t.Fatalf("draft=%+v", draft)
	}
}

func TestFromAuditPublishesDefinitionFieldUpdate(t *testing.T) {
	draft, publish, err := FromAudit("definition.field_updated", "resource_field", "field-1")
	if err != nil || !publish {
		t.Fatalf("FromAudit publish=%v err=%v", publish, err)
	}
	if draft.Type != "tenancit.resource_field.updated" || draft.AggregateID != "field-1" {
		t.Fatalf("draft=%+v", draft)
	}
}
