# AlemLive Backend

Go API для LiveKit токенов, Keycloak auth, AI-чата, STT и отчётов после встреч.

## Локальный запуск

```powershell
Copy-Item .env.example .env
go run ./cmd/server
```

Backend слушает `http://localhost:8080`. В полном Docker stack он доступен с хоста на `http://localhost:8088`.

## LiveKit + Egress + MinIO

В полном `docker compose up -d --build` backend уже получает рабочие значения:

- `LIVEKIT_URL=ws://livekit:7880` для backend и Egress внутри Docker.
- `LIVEKIT_PUBLIC_URL=ws://localhost:7880` для браузера.
- `LIVEKIT_EGRESS_ENABLED=true`.
- `LIVEKIT_EGRESS_AUDIO_ONLY=false`, чтобы сохранялся MP4 с видео, а не только аудио.
- `LIVEKIT_EGRESS_PUBLIC_BASE_URL=http://localhost:9000/alemlive-recordings`.
- `LIVEKIT_S3_ENDPOINT=http://minio:9000`.
- `LIVEKIT_S3_BUCKET=alemlive-recordings`.

Если backend запускается через `go run`, а LiveKit/Egress/MinIO в Docker, используй host-reachable адреса:

```powershell
$env:LIVEKIT_URL="ws://localhost:7880"
$env:LIVEKIT_PUBLIC_URL="ws://localhost:7880"
$env:LIVEKIT_EGRESS_WEBHOOK_URL="http://host.docker.internal:8080/api/livekit/webhook"
$env:LIVEKIT_S3_ENDPOINT="http://localhost:9000"
go run ./cmd/server
```

## AI/STT

- `LLM_BASE_URL` defaults to `https://llm.nitec.kz`.
- `LLM_API_KEY` enables AI chat and report analysis.
- `LLM_MODEL` defaults to `moonshotai/Kimi-K2.6`.
- `STT_BASE_URL` defaults to `LLM_BASE_URL`.
- `STT_API_KEY` defaults to `LLM_API_KEY`.
- `STT_MODEL` defaults to `openai/whisper-large-v3`.
- `STT_TIMEOUT_SECONDS` defaults to `900`.

## Storage

- `REPORTS_STORAGE_PATH` stores generated reports.
- `RECORDINGS_STORAGE_PATH` stores manually uploaded recordings.
- LiveKit Egress recordings are stored in MinIO bucket `alemlive-recordings`.

Keep `LIVEKIT_API_SECRET`, `LLM_API_KEY`, and `STT_API_KEY` only on the backend.
