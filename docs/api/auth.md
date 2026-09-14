# Auth

Session tokens are opaque in this contract. Authn is **not implemented**.

## POST /api/v1/auth/login

**Purpose.** Exchange email and password for a session token.

**Auth.** None.

**Request.**

```json
{
  "email": "sam.okonkwo@sentinel.dev",
  "password": "string"
}
```

**Validation.** `email` required, lowercase; `password` required, 1–256 chars.

**Response `200`.**

```json
{
  "data": {
    "token": "opaque-session-token",
    "user": {
      "id": "11111111-1111-4111-8111-111111111113",
      "email": "sam.okonkwo@sentinel.dev",
      "display_name": "Sam Okonkwo",
      "role": "responder"
    }
  }
}
```

**Status.** `200` success · `400` validation · `401` bad credentials or disabled · `429` login rate limit.

## POST /api/v1/auth/logout

**Purpose.** Invalidate the current session.

**Auth.** Bearer required.

**Request.** Empty body.

**Response `204`.** No body.

**Status.** `204` · `401`.

## GET /api/v1/me

**Purpose.** Return the authenticated user.

**Auth.** Bearer required.

**Response `200`.**

```json
{
  "data": {
    "id": "11111111-1111-4111-8111-111111111113",
    "email": "sam.okonkwo@sentinel.dev",
    "display_name": "Sam Okonkwo",
    "role": "responder",
    "status": "active"
  }
}
```

**Status.** `200` · `401`.
