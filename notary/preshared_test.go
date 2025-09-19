package notary_test

import (
	"testing"

	"github.com/james-lawrence/bw/notary"
)

func TestPresharedKeyCredentials(t *testing.T) {
	presharedKey := "test-secret-key"
	agentID := "test-agent-001"

	fingerprint, pubKey, err := notary.GeneratePresharedKeyCredentials(presharedKey, agentID)
	if err != nil {
		t.Fatalf("Failed to generate preshared key credentials: %v", err)
	}

	if fingerprint == "" {
		t.Error("Expected non-empty fingerprint")
	}

	if len(pubKey) == 0 {
		t.Error("Expected non-empty public key")
	}

	fingerprint2, pubKey2, err := notary.GeneratePresharedKeyCredentials(presharedKey, agentID)
	if err != nil {
		t.Fatalf("Failed to generate preshared key credentials second time: %v", err)
	}

	if fingerprint != fingerprint2 {
		t.Errorf("Expected identical fingerprints, got %s vs %s", fingerprint, fingerprint2)
	}

	if string(pubKey) != string(pubKey2) {
		t.Error("Expected identical public keys")
	}

	// Different agent ID should produce different credentials
	fingerprint3, _, err := notary.GeneratePresharedKeyCredentials(presharedKey, "different-agent")
	if err != nil {
		t.Fatalf("Failed to generate preshared key credentials for different agent: %v", err)
	}

	if fingerprint == fingerprint3 {
		t.Error("Expected different fingerprints for different agent IDs")
	}
}

func TestPresharedKeySigner(t *testing.T) {
	presharedKey := "test-secret-key"
	agentID := "test-agent-001"

	signer, err := notary.NewAgentPresharedKeySigner("/tmp/test", presharedKey, agentID)
	if err != nil {
		t.Fatalf("Failed to create preshared key signer: %v", err)
	}

	token, err := signer.Token()
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if token == "" {
		t.Error("Expected non-empty token")
	}
}