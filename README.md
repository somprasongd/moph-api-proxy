# MOPH Proxy API

Proxy service that manages authentication against the MOPH IC/FDH platforms, caches issued tokens, and forwards requests to the target APIs.

## Reference Documentation

- MOPH Immunization Center API – <https://docs.google.com/document/d/1Inyhfrte0pECsD8YoForTL2W8B2hOxezf0GpTGEjJr8/edit>
- EPIDEM Center API – <https://ddc.moph.go.th/viralpneumonia/file/g_surveillance/g_api_epidem_0165.pdf>
- MOPH-PHR API – <https://docs.google.com/document/d/1ZWCBJnxVCtjqmBGNjj1sLnYv11dVzUvnweui-26NDJ0/edit>
- MOPH Claim–NHSO (DMHT/EPI/dT Services) – <https://docs.google.com/document/d/1iiybB2y7NJkEhXTdS4DbYe3-Fs7aka7MlEns81lzODQ/edit>
- Financial Data Hub (FDH) – <https://drive.google.com/file/d/17XqRmSEOnXoJVwzmCwteuVdy-Gp_SUyW>
- Minimal Data Set (FDH Reservation) – <https://docs.google.com/document/d/1yDwflxOG_EG9HkEWbewfk446kmkGQ_2KiJhyqg2iY2E>

## Environment Variables

Set these variables through `nodemon.json`, shell exports, or the `moph-api-proxy.env` file used in production.

| Name | Required | Default | Description |
| --- | --- | --- | --- |
| `MOPH_HCODE` | ✅ | – | Hospital code used in every request payload. The service refuses to start when missing. |
| `USE_API_KEY` | ❌ | `true` | Toggle API key validation middleware. Set to `false` only when another perimeter control exists. |
| `HTTP_TIMEOUT_MS` | ❌ | `30000` | Axios request timeout for upstream calls. Increase if PHR/IC endpoints routinely take longer than 15s. |
| `APP_PORT` | ❌ | `3000` | Port used by the Express server. |
| `REDIS_HOST` | ❌ | `localhost` | Hostname of the Redis instance for caching tokens/payloads. Leave empty to force the in-memory cache (tokens reset on restart). Use Redis for production or any multi-instance deployment so tokens are shared. |
| `REDIS_PORT` | ❌ | `6379` | Redis TCP port. |
| `REDIS_PASSWORD` | ❌ | empty | Optional Redis password. |
| `MOPH_IC_API` | ❌ | `https://cvp1.moph.go.th` | Base URL for standard MOPH IC requests. |
| `MOPH_IC_AUTH` | ❌ | `https://cvp1.moph.go.th` | Auth endpoint used to retrieve MOPH IC tokens. |
| `MOPH_IC_AUTH_SECRET` | ❌ | `$jwt@moph#` | Secret key for hashing the MOPH IC credentials before requesting a token. |
| `FDH_API` | ❌ | `https://fdh.moph.go.th` | Base URL for FDH data requests. |
| `FDH_AUTH` | ❌ | `https://fdh.moph.go.th` | Auth endpoint for FDH token generation. |
| `FDH_AUTH_SECRET` | ❌ | `$jwt@moph#` | Secret key for hashing FDH credentials. |
| `MOPH_CLAIM_API` | ❌ | `https://claim-nhso.moph.go.th` | Base URL for the Claim (NHOS) endpoints. |
| `MOPH_PHR_API` | ❌ | `https://phr1.moph.go.th` | Base URL for the PHR endpoints. |
| `EPIDEM_API` | ❌ | `https://epidemcenter.moph.go.th/epidem` | Base URL for Epidem Center calls. |

> For UAT, replace the defaults with the relevant UAT hosts (for example, `https://uat-fdh.inet.co.th` for `FDH_API`).

## Development Setup

1. Copy `nodemon.example.json` to `nodemon.json` and fill in your environment variables.
2. (Recommended) Start Redis so tokens are shared across restarts:

   ```bash
   docker compose up -d
   ```

   If Redis is not available, the application automatically falls back to the in-memory cache described below.
3. Install dependencies:

   ```bash
   npm install
   ```

4. Start the dev server with auto reload:

   ```bash
   npm run dev
   ```

## Cache Backend

- Redis is the primary cache used to store tokens and the hashed payloads required to refresh them. This allows multiple proxy instances to share the same credentials and survive restarts.
- When Redis is not configured or becomes unreachable, the service automatically switches to an in-memory `Map` that mimics the Redis API.
- The in-memory mode is suitable for local development or single-instance deployments only. Tokens are wiped on restart and are **not** shared across containers, so keep Redis enabled in production.

## Production Deployment

1. Create `moph-api-proxy.env` with the environment values required by your deployment:

   ```env
   APP_PORT=3000
   MOPH_HCODE=your-hcode
   USE_API_KEY=true
   MOPH_IC_AUTH_SECRET=replace_me
   FDH_AUTH_SECRET=replace_me
   REDIS_HOST=redis
   REDIS_PORT=6379
   ```

   Add or override any of the variables listed earlier (API endpoints, passwords, etc.) as needed.
2. Create the shared Docker network (if not already created):

   ```bash
   docker network create webproxy
   ```

3. Deploy with the provided Compose files:

   ```bash
   docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
   ```

4. After the container is running:
   - Visit `http://<server-ip>:<port>/change-password` to set the proxy username/password.
   - Visit `http://<server-ip>:<port>/api-key` to log in with your MOPH IC credentials and retrieve the generated API key.

## Using the Proxy

Invoke downstream endpoints through `/api/<endpoint>` and include the API key unless you disabled the middleware:

```text
http://localhost:9090/api/ImmunizationTarget?x-api-key=YOUR_API_KEY&cid=1659900783037

# Without API key validation
http://localhost:9090/api/ImmunizationTarget?cid=1659900783037
```

- Replace `<endpoint>` with the actual MOPH IC API path.
- Append any query parameters (`cid`, etc.) required by the upstream API.
- The proxy logs the generated API key at startup; you can also fetch it through the `/api-key` route mentioned earlier.
