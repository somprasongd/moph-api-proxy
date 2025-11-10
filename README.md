# MOPH Proxy API

## MOPH IC API Document

<https://docs.google.com/document/d/1Inyhfrte0pECsD8YoForTL2W8B2hOxezf0GpTGEjJr8/edit>

## Development

- Set env rename `nodemon.example.json` to `nodemon.json` and config with your data

```json
{
  "env": {
    "MOPH_HCODE": "your-hcode",
    "USE_API_KEY": "true_or_false_default_is_true"
  }
}
```

To edit the secret key for hash password, add env like this:

```diff
{
  "env": {
    "MOPH_HCODE": "your-hcode",
    "USE_API_KEY": "true_or_false_default_is_true"
+    "MOPH_IC_AUTH_SECRET": "new_ecret_key",
+    "FDH_AUTH_SECRET": "new_ecret_key",
  }
}
```

To edit the API destination URL, add env like this:

```diff
{
  "env": {
    "MOPH_HCODE": "your-hcode",
    "USE_API_KEY": "true_or_false_default_is_true"
+    "MOPH_CLAIM_API": "https://claim-nhso.moph.go.th", // UAT: "https://uat-moph-nhso.inet.co.th",
+    "MOPH_PHR_API": "https://phr1.moph.go.th",
+    "EPIDEM_API": "https://epidemcenter.moph.go.th/epidem",
+    "FDH_API": "https://fdh.moph.go.th", // UAT: https://uat-fdh.inet.co.th
+    "FDH_AUTH": "https://fdh.moph.go.th",
+    "MOPH_IC_API": "https://cvp1.moph.go.th",
+    "MOPH_IC_AUTH": "https://cvp1.moph.go.th",
  }
}
```

- Start redis server you can run

```bash
docker-compose up -d
```

- Install dependencies

```bash
npm install
```

- Run dev server

```bash
npm run dev
```

## Production

- Create `moph-api-proxy.env` file

```env
MOPH_HCODE=your-hcode
USE_API_KEY=true_or_false_default_is_true
```

To edit the secret key for hash password, add env like this:

```diff
{
  "env": {
    "MOPH_HCODE": "your-hcode",
    "USE_API_KEY": "true_or_false_default_is_true"
+    "MOPH_IC_AUTH_SECRET": "new_ecret_key",
+    "FDH_AUTH_SECRET": "new_ecret_key",
  }
}
```

To edit the API destination URL, add env like this:

```diff
{
  "env": {
    "MOPH_HCODE": "your-hcode",
    "USE_API_KEY": "true_or_false_default_is_true"
+    "MOPH_CLAIM_API": "https://claim-nhso.moph.go.th", // UAT: "https://uat-moph-nhso.inet.co.th",
+    "MOPH_PHR_API": "https://phr1.moph.go.th",
+    "EPIDEM_API": "https://epidemcenter.moph.go.th/epidem",
+    "FDH_API": "https://fdh.moph.go.th", // UAT: https://uat-fdh.inet.co.th
+    "FDH_AUTH": "https://fdh.moph.go.th",
+    "MOPH_IC_API": "https://cvp1.moph.go.th",
+    "MOPH_IC_AUTH": "https://cvp1.moph.go.th",
  }
}
```

- To deploy with this production Compose file you can run

```bash
# create docker network
docker network create webproxy

# deploy
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

- Add Username and Password

Go to `http://your-server-ip:port/change-password`

- Get API Key

Go to `http://your-server-ip:port/api-key` and login with MOPH IC username and password.

## How to use

Call api from `http://your-server-ip:port/api/XXX?x-api-key=moph-api-proxy-api-key&AAA=yyyy&BBB=zzzz`

If set `USE_API_KEY=false` call `http://your-server-ip:port/api/XXX?AAA=yyyy&BBB=zzzz`

```text
http://localhost:9090/api/ImmunizationTarget?x-api-key=ETB4VPB-HJ4MEFA-MJ356Q5-HC3B87B&cid=1659900783037

# OR

http://localhost:9090/api/ImmunizationTarget?cid=1659900783037
```

> XXX is MOPH IC api endpoint

> AAA, BBB is MOPH IC api query parameters

> moph-api-proxy get from log when start server
