package settings

import "testing"

func TestValidateRejectsUnknownSecretLikeAndCrossFieldValues(t *testing.T) {
	for _, updates := range []map[string]string{
		{"database_url": "postgres://secret"},
		{SessionAbsoluteHours: "1", SessionIdleMinutes: "120"},
		{DefaultLocale: "fr-FR"},
		{UsageRetentionMonths: "0"},
		{WebhookDeliveryRetentionDays: "120", OutboxEventRetentionDays: "90"},
	} {
		if _, err := Validate(updates, nil); err == nil {
			t.Fatalf("updates accepted: %v", updates)
		}
	}
}

func TestValidateNormalizesClosedSettings(t *testing.T) {
	got, err := Validate(map[string]string{
		SessionAbsoluteHours: " 12 ", SessionIdleMinutes: "60", DefaultLocale: "en-US",
	}, nil)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if got[SessionAbsoluteHours] != "12" || got[DefaultLocale] != "en-US" {
		t.Fatalf("normalized=%v", got)
	}
}

func TestDefinitionsAndDefaultsAreDefensiveCopies(t *testing.T) {
	definitions := Definitions()
	defaults := Defaults()
	definitions[0].Key = "mutated"
	defaults[DefaultLocale] = "mutated"
	if Definitions()[0].Key == "mutated" || Defaults()[DefaultLocale] == "mutated" {
		t.Fatal("registry leaked mutable state")
	}
}
