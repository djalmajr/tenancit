package idempotency

import "testing"

func TestFingerprintSeparatesOperationFromPayload(t *testing.T) {
	first := Fingerprint("create", []byte(`{"name":"first"}`))
	repeated := Fingerprint("create", []byte(`{"name":"first"}`))
	differentOperation := Fingerprint("rotate", []byte(`{"name":"first"}`))

	if first != repeated {
		t.Fatal("same operation and payload produced different fingerprints")
	}
	if first == differentOperation {
		t.Fatal("different operations produced the same fingerprint")
	}
}
