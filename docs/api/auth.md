# Auth

Signed session tokens are JWTs bound to `auth_sessions` in PostgreSQL. The control plane accepts either:

```
Authorization: Bearer <token>
```

or the HttpOnly cookie `sentinel_session` (same token value). Browser traffic should use the Next.js same-origin proxy, which stores the token in that cookie and strips it from the login JSON body.

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

Successful login also sets `Set-Cookie: sentinel_session=...; HttpOnly; Path=/; SameSite=Lax`. Direct API clients may keep using `data.token`. The operator console never exposes that token to page JavaScript.

## POST /api/v1/auth/logout

**Purpose.** Invalidate the current session.

**Auth.** Bearer or session cookie.

**Request.** Empty body.

**Response `204`.** No body. The session cookie is cleared.

**Status.** `204` · `401`.

## GET /api/v1/me

**Purpose.** Return the authenticated user.

**Auth.** Bearer or session cookie.

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
