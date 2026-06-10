# AlemLive Backend

Go API for issuing short-lived LiveKit access tokens for the AlemLive React app.

## Start local LiveKit

From the repository root:

```powershell
docker compose -f docker-compose.livekit.yml up -d
```

The local dev LiveKit server listens at `ws://localhost:7880` and uses `devkey` / `devsecret-local-change-me-32-chars`.

## Run locally

```powershell
cd backend
Copy-Item .env.example .env
go run ./cmd/server
```

PowerShell can also pass variables for one session:

```powershell
$env:LIVEKIT_URL="wss://your-project.livekit.cloud"
$env:LIVEKIT_API_KEY="your_key"
$env:LIVEKIT_API_SECRET="your_secret"
go run ./cmd/server
```

The backend listens on `http://localhost:8080` by default. Vite proxies `/api` to this backend during development.

## Endpoints

- `GET /healthz` returns backend health.
- `GET /api/config` returns the configured LiveKit URL and token endpoint.
- `POST /api/livekit/token` accepts:

```json
{
  "roomName": "alem-meeting",
  "userName": "Madi"
}
```

Response:

```json
{
  "serverUrl": "wss://your-project.livekit.cloud",
  "token": "eyJ...",
  "roomName": "alem-meeting",
  "userName": "Madi",
  "expiresAt": "2026-06-10T12:00:00Z"
}
```

Keep `LIVEKIT_API_SECRET` only on the backend. Never expose it through Vite environment variables or browser code.

- `GET /api/meetings/analysis?roomName=...` returns a demo post-meeting report (summary, action items, transcript, insights, highlights, chapters).
- `GET /api/ask-ai` returns `{"url": "https://alem-workspace.gov.kz/web/alem-rag"}` — used by the "Спросить AI" button to open the Alem Workspace RAG assistant.
