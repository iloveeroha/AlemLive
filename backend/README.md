# AlemLive Backend

Go API for AlemLive: Keycloak auth, LiveKit tokens, meeting events, LiveKit Egress webhooks, reports, STT, diarization and AI chat.

## Local Run

```powershell
Copy-Item .env.example .env
go run ./cmd/server
```

In full Docker Compose the backend is available at `http://localhost:8088`.

## Main Endpoints

- `GET /healthz`
- `GET /api/config`
- `GET /api/auth/config`
- `POST /api/auth/token`
- `POST /api/livekit/token`
- `POST /api/meetings/events`
- `GET /api/reports`
- `GET /api/reports/{id}`
- `POST /api/reports/upload`
- `POST /api/reports/{id}/chat`
- `POST /api/livekit/webhook`

## AI/STT

The backend uses OpenAI-style endpoints:

- Chat: `POST {LLM_BASE_URL}/v1/chat/completions`
- STT: `POST {STT_BASE_URL}/v1/audio/transcriptions`

Defaults:

```env
LLM_BASE_URL=https://llm.nitec.kz
DEFAULT_LLM_MODEL=openai/gpt-oss-120b
STT_BASE_URL=https://llm.nitec.kz
DEFAULT_STT_MODEL=openai/whisper-large-v3
```

`LLM_MODEL` and `STT_MODEL` override the defaults.

## LiveKit/Egress

Inside Docker:

```env
LIVEKIT_URL=ws://livekit:7880
LIVEKIT_EGRESS_ENABLED=true
LIVEKIT_EGRESS_AUDIO_ONLY=false
LIVEKIT_S3_ENDPOINT=http://minio:9000
LIVEKIT_S3_BUCKET=alemlive-recordings
LIVEKIT_EGRESS_WEBHOOK_URL=http://backend:8080/api/livekit/webhook
```

The browser should use the URL returned by `/api/config` or `/api/livekit/token`. In the default stack it is proxied as `wss://<frontend-host>/livekit`.

For browser usage through the Docker frontend, keep `LIVEKIT_PUBLIC_URL` empty. If it is set to a direct `ws://host:7880` URL while the frontend is opened over HTTPS, browsers can block or downgrade the connection.

## Diarization

The optional bundled service uses pyannote:

```env
DIARIZATION_BASE_URL=http://diarization:8091
HF_TOKEN=your-huggingface-token
```

The service returns voice labels such as `SPEAKER_00`. Backend maps them to transcript lines as `Speaker 1`, `Speaker 2`; if participant names are known or speakers introduce themselves, backend can replace generic labels with names. Check `GET http://localhost:8091/healthz`; `configured: false` means `HF_TOKEN` is missing, invalid, or the pyannote model terms were not accepted.

## Report Statuses

Reports keep coarse `processingState` plus pipeline fields:

- `recordingStatus`: `missing`, `pending`, `running`, `completed`, `failed`
- `transcriptionStatus`: `not_started`, `pending`, `completed`, `failed`, `not_configured`
- `analysisStatus`: `not_started`, `pending`, `completed`, `failed`

The report is still saved when a later pipeline step fails.

## Tests

```powershell
$env:GOCACHE = Join-Path (Get-Location) '.gocache'
go test ./...
Remove-Item .gocache -Recurse -Force
```
