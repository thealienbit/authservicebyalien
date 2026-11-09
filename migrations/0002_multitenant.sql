-- =========================
-- APPS
-- =========================
CREATE TABLE apps (
    id              UUID PRIMARY KEY,
    name            TEXT NOT NULL,
    key_hash        TEXT NOT NULL UNIQUE,
    status          TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'DISABLED')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- =========================
-- USERS: tenancy + platform flag
-- =========================
-- app_id is NULL only for platform admins (they are not scoped to an app).
ALTER TABLE users
    ADD COLUMN app_id UUID REFERENCES apps(id) ON DELETE CASCADE,
    ADD COLUMN is_platform_admin BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX idx_users_app_id ON users(app_id);

-- =========================
-- ORGANIZATIONS
-- =========================
CREATE TABLE organizations (
    id              UUID PRIMARY KEY,
    app_id          UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'DISABLED')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_organizations_app_id ON organizations(app_id);

-- =========================
-- MEMBERSHIPS
-- =========================
CREATE TABLE memberships (
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    role            TEXT NOT NULL CHECK (role IN ('SUPERADMIN', 'ADMIN', 'CLERK')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, organization_id)
);

CREATE INDEX idx_memberships_organization_id ON memberships(organization_id);

-- =========================
-- USER EMAILS: scope to app
-- =========================
-- Denormalized from users.app_id. The write path must set both.
ALTER TABLE user_emails
    ADD COLUMN app_id UUID REFERENCES apps(id) ON DELETE CASCADE;

ALTER TABLE user_emails DROP CONSTRAINT user_emails_email_key;

-- Emails are unique per app.
CREATE UNIQUE INDEX idx_user_emails_app_email ON user_emails(app_id, email);

-- Platform admins and any other app-less email must still be unique. A plain
-- UNIQUE (app_id, email) does not catch NULL app_id rows.
CREATE UNIQUE INDEX idx_user_emails_platform_email ON user_emails(email) WHERE app_id IS NULL;
