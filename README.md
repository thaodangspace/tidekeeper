# Tidekeepers

Tidekeepers is a full-stack game app in one project. The SvelteKit client and
Deno API live together at the repository root; API implementation and tests are
under `server/`.

Players enter a display name to create or resume a guest player. The API derives
an opaque player identity from the normalized name and effective client IP, then
uses an HttpOnly session cookie for gameplay. No credential account is required.

## Development

Install frontend dependencies and start the client:

```sh
npm install
npm run dev
```

The client proxies `/auth` and `/api` to `http://localhost:8089`. Start the API
in a second terminal:

```sh
deno task dev
```

Or use the package script:

```sh
npm run api:dev
```

## Production / Deno Deploy

Build the static client and serve it from the same Deno process and port:

```sh
npm start
```

For Deno Deploy, use `npm run build` as the build command and
`server/main.ts` (or `deno task start`) as the entrypoint. Set `APP_ENV=production`
and the required production secrets, including `PLAYER_ID_SECRET`. The app
listens on `PORT` (or `8000` by default) and serves both the UI and API.

The server uses the direct Deno connection address by default. Set
`TRUSTED_PROXY=true` only when a deployment proxy overwrites the configured
`TRUSTED_PROXY_HEADER` (default `x-forwarded-for`) with a trusted client IP;
forwarding headers are otherwise ignored.

## Checks

```sh
npm run check
deno task fmt:check
deno task lint
deno task check
deno task test
```

Build the static client with an API origin when deploying separately:

```sh
PUBLIC_API_BASE_URL=https://api.example.com npm run build
```
