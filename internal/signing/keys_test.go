package signing

import (
	"crypto/rand"
	"encoding/base64"
	"testing"
)

func TestThumbprintStableAndUnique(t *testing.T) {
	first, err := Generate()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	thumbprint, err := Thumbprint(first.PublicKey)
	if err != nil {
		t.Fatalf("thumbprint: %v", err)
	}
	if thumbprint != first.KeyID {
		t.Fatalf("thumbprint %q != kid %q", thumbprint, first.KeyID)
	}

	second, err := Generate()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if second.KeyID == first.KeyID {
		t.Fatal("two generated keys share a kid")
	}
}

func TestCipherRoundTrip(t *testing.T) {
	kek := make([]byte, 32)
	if _, err := rand.Read(kek); err != nil {
		t.Fatal(err)
	}

	cipher, err := NewCipher(base64.StdEncoding.EncodeToString(kek))
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}

	encrypted, err := cipher.Encrypt([]byte("signing key material"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	decrypted, err := cipher.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(decrypted) != "signing key material" {
		t.Fatalf("round trip mismatch: %q", decrypted)
	}

	other := make([]byte, 32)
	if _, err := rand.Read(other); err != nil {
		t.Fatal(err)
	}
	wrongCipher, err := NewCipher(base64.StdEncoding.EncodeToString(other))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wrongCipher.Decrypt(encrypted); err == nil {
		t.Fatal("decrypt with the wrong KEK succeeded")
	}
}

func TestNewCipherRejectsWrongKeyLength(t *testing.T) {
	if _, err := NewCipher(base64.StdEncoding.EncodeToString([]byte("too short"))); err == nil {
		t.Fatal("expected an error for a non-32-byte KEK")
	}
}
