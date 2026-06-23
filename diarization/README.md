# AlemLive diarization service

This optional service separates speakers with `pyannote.audio` and exposes a small HTTP API for the Go backend.

## Requirements

1. Create a Hugging Face token.
2. Accept access terms for `pyannote/speaker-diarization-3.1`.
3. Put the token into the root `.env` or your shell:

```env
HF_TOKEN=
DIARIZATION_BASE_URL=http://diarization:8091
```

## Run

```powershell
docker compose up -d --build diarization backend
```

The backend sends recordings to `POST /diarize` and expects:

```json
{
  "segments": [
    { "start": 0.0, "end": 3.2, "speaker": "SPEAKER_00" },
    { "start": 3.2, "end": 8.1, "speaker": "SPEAKER_01" }
  ]
}
```

The Go backend maps these segments onto STT transcript lines and exposes them as `Speaker 1`, `Speaker 2`, and so on.
