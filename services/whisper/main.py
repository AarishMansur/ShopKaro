import io
import os
import tempfile

from fastapi import FastAPI, File, HTTPException, UploadFile
from faster_whisper import WhisperModel

MODEL_NAME = os.getenv("WHISPER_MODEL", "base")
LANGUAGE = os.getenv("WHISPER_LANGUAGE") or None

app = FastAPI(title="shopkaro-whisper")
_model: WhisperModel | None = None


def model() -> WhisperModel:
    global _model
    if _model is None:
        _model = WhisperModel(MODEL_NAME, device="cpu", compute_type="int8")
    return _model


@app.get("/healthz")
def healthz() -> dict:
    return {"ok": True, "model": MODEL_NAME}


@app.post("/transcribe")
async def transcribe(file: UploadFile = File(...)) -> dict:
    data = await file.read()
    if not data:
        raise HTTPException(status_code=400, detail="empty audio upload")

    suffix = os.path.splitext(file.filename or "")[1] or ".webm"
    tmp = tempfile.NamedTemporaryFile(delete=False, suffix=suffix)
    try:
        tmp.write(data)
        tmp.close()
        segments, _info = model().transcribe(tmp.name, language=LANGUAGE, vad_filter=True)
        text = " ".join(seg.text.strip() for seg in segments).strip()
    finally:
        try:
            os.unlink(tmp.name)
        except OSError:
            pass

    if not text:
        raise HTTPException(status_code=422, detail="no speech detected")
    return {"text": text}
