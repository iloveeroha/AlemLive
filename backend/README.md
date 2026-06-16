# AlemLive Backend

Go API for issuing short-lived LiveKit access tokens and AI-backed meeting assistance for the AlemLive React app.

## Start Local LiveKit

From the repository root:

```powershell
docker compose -f docker-compose.livekit.yml up -d
```

The local dev LiveKit server listens at `ws://localhost:7880` and uses `devkey` / `devsecret-local-change-me-32-chars`.

## Run Locally

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
$env:LLM_API_KEY="your_llm_key"
$env:LLM_MODEL="moonshotai/Kimi-K2.6"
$env:STT_MODEL="openai/whisper-large-v3"
go run ./cmd/server
```

The backend listens on `http://localhost:8080` by default. Vite proxies `/api` to this backend during development.

## Configuration

- `PORT` defaults to `8080`.
- `LIVEKIT_URL`, `LIVEKIT_API_KEY`, and `LIVEKIT_API_SECRET` configure LiveKit token issuing.
- `TOKEN_TTL_MINUTES` defaults to `120`.
- `ALLOWED_ORIGINS` is a comma-separated CORS allowlist.
- `LLM_BASE_URL` defaults to `https://llm.nitec.kz`.
- `LLM_API_KEY` enables AI chat and AI meeting analysis.
- `LLM_MODEL` defaults to `moonshotai/Kimi-K2.6`.
- `STT_BASE_URL` defaults to `LLM_BASE_URL` and must expose OpenAI-compatible `/v1/audio/transcriptions`.
- `STT_API_KEY` defaults to `LLM_API_KEY`.
- `STT_MODEL` defaults to `openai/whisper-large-v3` for the NITEC OpenAI-compatible gateway.
- `STT_TIMEOUT_SECONDS` defaults to `900` for longer audio/video transcription jobs.
- `LLM_TIMEOUT_SECONDS` defaults to `60`.
- `REPORTS_STORAGE_PATH` defaults to `data/reports.json` and stores generated meeting reports between backend restarts.
- `RECORDINGS_STORAGE_PATH` defaults to `data/recordings` and stores uploaded meeting audio/video files for report playback.

Keep `LIVEKIT_API_SECRET`, `LLM_API_KEY`, and `STT_API_KEY` only on the backend. Never expose them through Vite environment variables or browser code.

## Endpoints

- `GET /healthz` returns backend health.
- `GET /api/config` returns public frontend configuration.
- `GET /api/ai/status` returns whether AI is configured plus the active public model metadata.
- `POST /api/ai/chat` sends a report-aware chat prompt to the configured LLM provider.
- `POST /api/meetings/transcribe` accepts audio/video upload or `transcriptText`, runs Whisper STT, then returns transcript plus meeting analysis.
- `GET /api/meetings/analysis?roomName=...` returns AI-generated meeting analysis when AI is configured, otherwise falls back to a demo report.
- `GET /api/ask-ai` returns `{"url": "https://alem-workspace.gov.kz/web/alem-rag"}` for the external Alem Workspace RAG assistant.
- `POST /api/livekit/token` accepts:

```json
{
  "roomName": "alem-meeting",
  "userName": "Madi",
  "isHost": true
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
