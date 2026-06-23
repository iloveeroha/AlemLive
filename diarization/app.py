import os
import subprocess
import tempfile
from pathlib import Path
from typing import Optional

from fastapi import FastAPI, File, Form, HTTPException, UploadFile
from pyannote.audio import Pipeline


MODEL_NAME = os.getenv("DIARIZATION_MODEL", "pyannote/speaker-diarization-3.1")
HF_TOKEN = os.getenv("HF_TOKEN") or os.getenv("HUGGINGFACE_TOKEN")

app = FastAPI(title="AlemLive Diarization")
pipeline: Optional[Pipeline] = None


@app.on_event("startup")
def load_pipeline() -> None:
    global pipeline
    if not HF_TOKEN:
        return
    pipeline = Pipeline.from_pretrained(MODEL_NAME, use_auth_token=HF_TOKEN)


@app.get("/healthz")
def healthz() -> dict:
    return {
        "status": "ok" if pipeline is not None else "missing_hf_token",
        "model": MODEL_NAME,
        "configured": pipeline is not None,
    }


@app.post("/diarize")
async def diarize(
    file: UploadFile = File(...),
    participants: str = Form(""),
    min_speakers: Optional[int] = Form(None),
    max_speakers: Optional[int] = Form(None),
) -> dict:
    if pipeline is None:
        raise HTTPException(
            status_code=503,
            detail="Diarization model is not loaded. Set HF_TOKEN and accept the pyannote model terms.",
        )

    suffix = Path(file.filename or "recording.wav").suffix.lower() or ".wav"
    with tempfile.NamedTemporaryFile(delete=False, suffix=suffix) as tmp:
        tmp.write(await file.read())
        tmp_path = tmp.name
    audio_path = tmp_path

    try:
        if suffix not in {".wav", ".flac", ".ogg", ".aiff", ".aif"}:
            audio_path = convert_to_wav(tmp_path)

        kwargs = {}
        if min_speakers:
            kwargs["min_speakers"] = min_speakers
        if max_speakers:
            kwargs["max_speakers"] = max_speakers

        diarization = pipeline(audio_path, **kwargs)
        segments = []
        for turn, _, speaker in diarization.itertracks(yield_label=True):
            segments.append(
                {
                    "start": float(turn.start),
                    "end": float(turn.end),
                    "speaker": speaker,
                }
            )

        return {
            "segments": segments,
            "participants": participants,
            "model": MODEL_NAME,
        }
    finally:
        try:
            os.remove(tmp_path)
        except OSError:
            pass
        if audio_path != tmp_path:
            try:
                os.remove(audio_path)
            except OSError:
                pass


def convert_to_wav(path: str) -> str:
    fd, wav_path = tempfile.mkstemp(suffix=".wav")
    os.close(fd)
    try:
        subprocess.run(
            [
                "ffmpeg",
                "-y",
                "-i",
                path,
                "-vn",
                "-ac",
                "1",
                "-ar",
                "16000",
                "-f",
                "wav",
                wav_path,
            ],
            check=True,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.PIPE,
            text=True,
        )
    except subprocess.CalledProcessError as exc:
        try:
            os.remove(wav_path)
        except OSError:
            pass
        detail = (exc.stderr or "ffmpeg could not extract audio").strip()
        raise HTTPException(status_code=422, detail=detail[-800:]) from exc
    return wav_path
