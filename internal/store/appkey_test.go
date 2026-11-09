package store

import "testing"

func TestGenerateAppKey(t *testing.T) {
	plain, hash, err := GenerateAppKey()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if len(plain) != 43 {
		t.Fatalf("expected 43 base64url chars for 32 bytes, got %d", len(plain))
	}
	if hash != HashAppKey(plain) {
		t.Fatal("hash does not match HashAppKey(plain)")
	}

	other, _, err := GenerateAppKey()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if plain == other {
		t.Fatal("two generated keys are identical")
	}
}

func TestHashAppKeyIsDeterministic(t *testing.T) {
	if HashAppKey("abc") != HashAppKey("abc") {
		t.Fatal("HashAppKey is not deterministic")
	}
	if HashAppKey("abc") == HashAppKey("abd") {
		t.Fatal("HashAppKey collided")
	}
}
