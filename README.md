# Tidekeepers

Tidekeepers is a full-stack game app in one project. The SvelteKit client and
Deno API live together at the repository root; API implementation and tests are
under `server/`.

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
and the required production secrets. The app listens on `PORT` (or `8000` by
default) and serves both the UI and API.

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
