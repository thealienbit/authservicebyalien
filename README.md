Overview of tasks

- [x] Add HTTP router (chi) 
- [x] Add /auth/v1/register endpoint
- [x] Add password hashing (Argon2id)
- [] Implement user + email persistence
- [] Email verification flow
- [] JWT issuing + JWKS

Remember to add 
CORS, 
rate limiting, or 
auth middleware yet — we’ll add them intentionally later.

Implement /auth/v1/register
What this endpoint will do
Validate request
Hash password (Argon2id)
Create:
users
user_emails
password_credentials
Do everything in one DB transaction
Return generic success (no email enumeration)
Email sending will be stubbed for now.

Later we can move keys to:
KMS
HSM
Vault

