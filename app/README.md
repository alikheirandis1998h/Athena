# base

A reusable base project for building new products on: a Go/PostgreSQL backend and an Expo/React
Native client, wired together with JWT access/refresh authentication. Use this as the starting
point for a new app — rename the module/package names, keep the plumbing.

## Structure

- `backend/` — Go + Gin API, PostgreSQL via sqlc/pgx, golang-migrate migrations, zerolog logging,
  Swagger docs, JWT auth (access + refresh with rotation). See `backend/` (Makefile has all
  common tasks).
- The repo root (this folder) — Expo + TypeScript client using Expo Router, axios with auto token
  refresh, and expo-secure-store for token persistence. Targets **iOS, Android, and web from one
  codebase** (via `react-native-web`) — there's no separate website project needed unless you want
  a distinct marketing site. Screens live in `routes/`.

## Running locally

Backend (requires Docker for Postgres, or point `DATABASE_URL` at your own instance):

```bash
cd backend
cp .env.example .env
make docker-up        # starts Postgres (and the API, if you build the image)
make migrate-up
make run              # serves on :8080, Swagger UI at /swagger/index.html
```

Client app (iOS/Android/web), from this folder:

```bash
cp .env.example .env  # points at http://localhost:8080 by default
npm install
npx expo start        # press i / a / w for iOS, Android, or web
```

## Auth model

- Passwords hashed with bcrypt.
- Access tokens: short-lived JWT (15m default), signed with `JWT_ACCESS_SECRET`.
- Refresh tokens: longer-lived JWT (7d default), signed with a separate `JWT_REFRESH_SECRET`,
  hashed and stored server-side so they can be looked up and revoked. Every refresh call rotates
  the token (old one revoked, new one issued) so a stolen refresh token has a single use window.
- The client attaches the access token to every request and silently refreshes once on a
  401, queuing concurrent requests behind the single in-flight refresh call.

## Regenerating backend codegen

`sqlc` (DB layer) and `swag` (Swagger docs) are checked in as generated output but are meant to be
regenerated as you change queries/handlers:

```bash
cd backend
make sqlc-generate   # after editing queries/*.sql or migrations/*.sql
make swag-init        # after editing handler @-annotations
```
