-- Persisted JWT signing keys. Previously regenerated on every boot (main.go),
-- which invalidated all access tokens and defeated JWKS caching downstream.
CREATE TABLE signing_keys (
    id                    UUID PRIMARY KEY,
    -- RFC 7638 JWK thumbprint. A new key must produce a new kid so JWKS
    -- consumers see a cache miss instead of silently reusing a stale key.
    kid                   TEXT NOT NULL UNIQUE,
    alg                   TEXT NOT NULL DEFAULT 'RS256' CHECK (alg IN ('RS256')),
    -- PKCS#8 PEM, AES-GCM encrypted with a KEK supplied via env.
    private_key_encrypted TEXT NOT NULL,
    is_active             BOOLEAN NOT NULL DEFAULT TRUE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    retired_at            TIMESTAMPTZ
);

-- Exactly one key may be used for signing at a time. Older keys stay
-- non-active and non-retired so JWKS can continue to publish them.
CREATE UNIQUE INDEX idx_signing_keys_one_active
    ON signing_keys (is_active)
    WHERE is_active AND retired_at IS NULL;
