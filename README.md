# AlemLive

AlemLive is a local meeting stack: React frontend, Go backend, Keycloak auth, LiveKit video rooms, LiveKit Egress recording, MinIO/S3 storage, STT, diarization and AI reports.

## Quick Start on Windows

```powershell
Copy-Item .env.example .env
Copy-Item backend/.env.example backend/.env
docker compose up -d --build
```

Open:

- Frontend: `https://localhost:5174` or `http://localhost:5173`
- Backend: `http://localhost:8088`
- Keycloak: `http://localhost:8080`
- MinIO console: `http://localhost:9001`
- LiveKit websocket is proxied through frontend as `/livekit`

The frontend uses a self-signed HTTPS certificate. The browser will ask you to accept it once. Camera and microphone work on `https://localhost:5174`; for another computer on the LAN, set your current IP in `.env` and rebuild.

## LAN Testing

Find your Windows IPv4:

```powershell
ipconfig
```

Then edit root `.env`:

```env
APP_HOST=YOUR_LAN_IP
LIVEKIT_NODE_IP=YOUR_LAN_IP
KEYCLOAK_PUBLIC_URL=http://YOUR_LAN_IP:8080
LIVEKIT_EGRESS_PUBLIC_BASE_URL=http://YOUR_LAN_IP:9000/alemlive-recordings
```

Rebuild after changing `APP_HOST`, because it is used in the frontend HTTPS certificate:

```powershell
docker compose up -d --build
```

For Keycloak redirects, keep `keycloak/alemlive-realm.json` in sync with the current LAN IP or update the `alemlive` client from the Keycloak admin console. This dev setup imports the realm on container creation; if Keycloak is recreated, local test users may need to be registered again.

## Required Env

Root `.env` controls Docker Compose and public URLs:

- `APP_HOST`
- `LIVEKIT_NODE_IP`
- `KEYCLOAK_PUBLIC_URL`
- `LIVEKIT_API_KEY`
- `LIVEKIT_API_SECRET`
- `MINIO_ROOT_USER`
- `MINIO_ROOT_PASSWORD`
- `MINIO_BUCKET`
- `LLM_API_KEY`
- `STT_API_KEY`
- `HF_TOKEN`
- `DIARIZATION_BASE_URL`

Backend `.env` controls local `go run` and backend defaults:

- `LLM_BASE_URL=https://llm.nitec.kz`
- `DEFAULT_LLM_MODEL=openai/gpt-oss-120b`
- `STT_BASE_URL=https://llm.nitec.kz`
- `DEFAULT_STT_MODEL=openai/whisper-large-v3`
- `REPORTS_STORAGE_PATH=data/reports.json`
- `RECORDINGS_STORAGE_PATH=data/recordings`

Do not commit real API keys, tokens or passwords.

## How the Report Pipeline Works

1. User creates or joins a LiveKit room.
2. Backend creates a meeting report immediately.
3. LiveKit Egress records the room.
4. Egress writes MP4/OGG to MinIO bucket.
5. LiveKit webhook notifies backend.
6. Backend downloads the recording from MinIO.
7. Backend sends audio/video to STT.
8. Backend sends audio/video to diarization.
9. Backend maps speaker segments to transcript lines.
10. Backend sends transcript to LLM for summary, action items, highlights and chapters.
11. Backend saves the JSON report in `backend/data/reports.json`.

If recording or transcription fails, the report is still saved with clear statuses.

## Checks

```powershell
docker compose ps
Invoke-RestMethod http://localhost:8088/healthz
Invoke-RestMethod http://localhost:8088/api/config
Invoke-RestMethod http://localhost:8091/healthz
```

MinIO:

- Console: `http://localhost:9001`
- Bucket: `alemlive-recordings`

Keycloak:

- Admin console: `http://localhost:8080`
- Realm: `alemlive`
- Client: `alemlive`

Go tests:

```powershell
cd backend
$env:GOCACHE = Join-Path (Get-Location) '.gocache'
go test ./...
Remove-Item .gocache -Recurse -Force
```

Frontend build:

```powershell
cd frontend
npm run build
```
