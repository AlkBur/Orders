# ADR-002: Initial Password Authentication

## Status

Accepted

## Context

The application needed a way for users without a password (newly created
accounts) to authenticate for the first time and set their own password.

The first attempt allowed login without any password check for users
with `NeedsPasswordSetup()`, creating a session before password setup.
This violated the principle that a session must only be created after
successful authentication.

The second attempt assigned a hardcoded default password (`pass123`) to
the admin account during seed. This worked but had two issues:

- The default password was not visible in the configuration.
- Only admin had a password; future users would still need a mechanism
  for first-time authentication.

## Decision

Introduce an Initial Password mechanism:

- Every user without a password authenticates through a shared Initial
  Password stored in the application configuration (`auth.initial_password`).
- A user with a password authenticates exclusively through their own
  password (bcrypt).
- After successful authentication through the Initial Password, the user
  is redirected to `/set-password` to create their own password.
- After setting the password, the session is destroyed and the user must
  log in again with their new password.
- The Initial Password no longer works for that account once a user
  password is set.
- The Initial Password is required at startup — the application fails
  with an error if it is not configured.

## Consequences

### Positive

- No authentication without credential verification.
- No hardcoded default secrets.
- The Initial Password is visible and configurable by the administrator.
- No separate invite infrastructure is needed for MVP.
- Works for any number of users without passwords.

### Negative

- All users without passwords share the same Initial Password until
  they set their own.
- A user who forgets their password cannot use the Initial Password
  (it is only for users with no password set).
- The Initial Password must be strong to reduce shared-secret risk.
