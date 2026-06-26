# AlemLive

AlemLive is a local meeting stack: React frontend, Go backend, Keycloak auth, LiveKit video rooms, LiveKit Egress recording, MinIO/S3 storage, STT, diarization and AI reports.

## Quick Start on Windows

```powershell
Copy-Item .env.example .env
Copy-Item backend/.env.example backend/.env
docker volume create alemlive_keycloak_data
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

Fast path on Windows:

```powershell
.\scripts\update-local-ip.ps1
```

The script finds the current LAN IPv4, updates `.env` and `backend/.env`, rebuilds the frontend HTTPS certificate, and recreates the Docker services that depend on public URLs. If automatic detection picks the wrong adapter, pass the IP manually:

```powershell
.\scripts\update-local-ip.ps1 -Ip 10.111.226.73
```

To only update env files without restarting Docker:

```powershell
.\scripts\update-local-ip.ps1 -NoDocker
```

Manual setup is also possible. Find your Windows IPv4:

```powershell
ipconfig
```

Then edit root `.env`:

```env
APP_HOST=YOUR_LAN_IP
LIVEKIT_NODE_IP=YOUR_LAN_IP
KEYCLOAK_PUBLIC_URL=http://YOUR_LAN_IP:8080
LIVEKIT_EGRESS_PUBLIC_BASE_URL=http://YOUR_LAN_IP:9000/alemlive-recordings
LIVEKIT_PUBLIC_URL=
```

Rebuild after changing `APP_HOST`, because it is used in the frontend HTTPS certificate:

```powershell
docker compose up -d --build
```

Leave `LIVEKIT_PUBLIC_URL` empty for browser usage through the frontend HTTPS proxy. Set it only when a client needs to connect directly to LiveKit, for example a mobile app.

For Keycloak redirects, keep `keycloak/alemlive-realm.json` in sync with the current LAN IP or update the `alemlive` client from the Keycloak admin console.

## Meeting Links

AlemLive rooms use Google Meet-style meeting codes:

- code format: `abc-defg-hij`
- share link format: `https://YOUR_HOST:5174/#meeting/abc-defg-hij`
- users may join by pasting either the code or the full link

The backend canonicalizes both the link and the code to the same LiveKit room name, so every participant who uses the same code joins the same room.

Keycloak users are stored in the external Docker volume `alemlive_keycloak_data`. Create it once with `docker volume create alemlive_keycloak_data`; after that users survive normal restarts and even `docker compose down -v`, because Compose does not own external volumes. Users are lost only if this volume is removed manually or Docker Desktop is reset.

To intentionally reset all local Keycloak users:

```powershell
docker compose stop keycloak backend frontend
docker volume rm alemlive_keycloak_data
docker volume create alemlive_keycloak_data
docker compose up -d keycloak backend frontend
```

## Required Env

Root `.env` controls Docker Compose and public URLs:

- `APP_HOST`
- `LIVEKIT_NODE_IP`
- `KEYCLOAK_PUBLIC_URL`
- `KEYCLOAK_ADMIN_USERNAME`
- `KEYCLOAK_ADMIN_PASSWORD`
- `KEYCLOAK_REALM`
- `KEYCLOAK_CLIENT_ID`
- `LIVEKIT_API_KEY`
- `LIVEKIT_API_SECRET`
- `MINIO_ROOT_USER`
- `MINIO_ROOT_PASSWORD`
- `MINIO_BUCKET`
- `LLM_API_KEY`
- `STT_API_KEY`
- `HF_TOKEN`
- `DIARIZATION_BASE_URL` (defaults to `http://diarization:8091` in Docker Compose)

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

## Speaker Attribution

Current default:

- `RECORDING_MODE=room_composite`
- STT creates transcript text and timestamps.
- Diarization/pyannote can split speakers as `Speaker 1`, `Speaker 2`, etc.
- Backend does not guess participant names from speaker order.

Participant names are used only when they come from a reliable source:

- Keycloak/local auth user profile.
- LiveKit identity/name/metadata.
- Participant audio-track recording.
- Manual speaker mapping.

Why this matters: room composite recording is one mixed audio/video file. With that file alone backend cannot prove that `Speaker 1` is Madi and `Speaker 2` is Asyl. If backend guessed, reports would look nicer but could be wrong.

Prepared env flags:

```env
RECORDING_MODE=room_composite
RECORDING_FALLBACK_MODE=room_composite
ENABLE_DIARIZATION_FALLBACK=true
ENABLE_SPEAKER_MANUAL_MAPPING=true
TRANSCRIPT_PREFER_PARTICIPANT_NAMES=true
```

For stronger speaker names, set:

```env
RECORDING_MODE=participant_tracks
RECORDING_FALLBACK_MODE=room_composite
```

In this mode backend starts a hybrid recording: room-composite MP4 stays available for report video playback, while each known participant microphone track is recorded separately as an audio file. STT runs on those audio tracks first and labels transcript lines by LiveKit participant identity/name. If audio track IDs are not available yet or track egress fails, backend falls back to the room-composite recording and then uses diarization.

## Checks

```powershell
docker compose ps
Invoke-RestMethod http://localhost:8088/healthz
Invoke-RestMethod http://localhost:8088/api/config
Invoke-RestMethod http://localhost:8088/api/diagnostics/recording
Invoke-RestMethod http://localhost:8091/healthz
```

`/healthz` for diarization returns `configured: false` when `HF_TOKEN` is missing, invalid, or the pyannote model terms were not accepted on Hugging Face.

`/api/diagnostics/recording` checks the recording chain without exposing secrets:

- LiveKit URL and credentials are present
- LiveKit Egress is configured
- webhook URL is present
- MinIO/S3 endpoint, bucket and credentials are present
- recording bucket is reachable
- STT, LLM and diarization are configured

If the report says "video unavailable" or "recording failed", check the backend diagnostics first, then container logs:

```powershell
docker compose logs --tail=120 backend
docker compose logs --tail=120 livekit-egress
docker compose logs --tail=120 livekit
docker compose logs --tail=120 minio-init
docker compose logs --tail=120 minio
```

For Docker Compose, backend and Egress use internal service names (`minio`, `backend`, `livekit`). Browser playback uses public URLs from `.env`, for example `LIVEKIT_EGRESS_PUBLIC_BASE_URL=http://localhost:9000/alemlive-recordings` or the current LAN IP.

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
