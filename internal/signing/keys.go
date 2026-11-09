package signing

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"

	"github.com/google/uuid"
)

const Algorithm = "RS256"

// Key is a loaded signing key. ID, KeyID, Algorithm and Active come from the
// signing_keys table; the RSA material is decrypted at load time.
type Key struct {
	ID         uuid.UUID
	KeyID      string
	Algorithm  string
	Active     bool
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

func Generate() (*Key, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("signing: generate rsa key: %w", err)
	}

	kid, err := Thumbprint(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}

	return &Key{
		KeyID:      kid,
		Algorithm:  Algorithm,
		Active:     true,
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}, nil
}

func (k *Key) Validate() error {
	if k == nil || k.PrivateKey == nil || k.PublicKey == nil {
		return errors.New("signing: invalid key")
	}
	if k.KeyID == "" {
		return errors.New("signing: missing kid")
	}
	return nil
}

// Thumbprint computes the RFC 7638 JWK thumbprint. Used as the kid so that a
// key change is always visible to JWKS consumers as a new kid.
func Thumbprint(publicKey *rsa.PublicKey) (string, error) {
	if publicKey == nil {
		return "", errors.New("signing: nil public key")
	}

	n := base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(publicKey.E)).Bytes())

	// RFC 7638 requires lexicographic member order: e, kty, n.
	canonical := fmt.Sprintf(`{"e":"%s","kty":"RSA","n":"%s"}`, e, n)
	sum := sha256.Sum256([]byte(canonical))

	return base64.RawURLEncoding.EncodeToString(sum[:]), nil
}

func (k *Key) JWK() map[string]string {
	n := base64.RawURLEncoding.EncodeToString(k.PublicKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(k.PublicKey.E)).Bytes())

	return map[string]string{
		"kty": "RSA",
		"kid": k.KeyID,
		"use": "sig",
		"alg": k.Algorithm,
		"n":   n,
		"e":   e,
	}
}

func (k *Key) MarshalPrivatePEM() ([]byte, error) {
	der, err := x509.MarshalPKCS8PrivateKey(k.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("signing: marshal private key: %w", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), nil
}

func ParsePrivatePEM(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("signing: no PEM block found")
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("signing: parse private key: %w", err)
	}

	privateKey, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("signing: not an RSA private key")
	}

	return privateKey, nil
}

// Active returns the key currently used for signing, or nil.
func Active(keys []*Key) *Key {
	for _, key := range keys {
		if key.Active {
			return key
		}
	}
	return nil
}

// Cipher encrypts signing keys at rest so a database dump alone cannot mint
// tokens. The key-encryption key is supplied out of band via JWT_KEK.
type Cipher struct {
	aead cipher.AEAD
}

func NewCipher(kekBase64 string) (*Cipher, error) {
	kek, err := base64.StdEncoding.DecodeString(kekBase64)
	if err != nil {
		return nil, fmt.Errorf("signing: decode JWT_KEK: %w", err)
	}
	if len(kek) != 32 {
		return nil, fmt.Errorf("signing: JWT_KEK must be 32 bytes, got %d", len(kek))
	}

	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, fmt.Errorf("signing: create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("signing: create gcm: %w", err)
	}

	return &Cipher{aead: aead}, nil
}

func (c *Cipher) Encrypt(plaintext []byte) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("signing: read nonce: %w", err)
	}

	out := make([]byte, 0, len(nonce)+len(plaintext)+c.aead.Overhead())
	out = append(out, nonce...)
	out = c.aead.Seal(out, nonce, plaintext, nil)

	return base64.StdEncoding.EncodeToString(out), nil
}

func (c *Cipher) Decrypt(encoded string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("signing: decode ciphertext: %w", err)
	}

	nonceSize := c.aead.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("signing: ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("signing: decrypt: %w", err)
	}

	return plaintext, nil
}
