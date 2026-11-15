import logging
import os
import tempfile
import threading

import essentia.standard as es
import numpy as np
from fastapi import FastAPI, File, HTTPException, UploadFile
from fastapi.responses import JSONResponse

logger = logging.getLogger("audio_processor")
logging.basicConfig(level=logging.INFO)

app = FastAPI()

# --- MAEST embedding configuration -----------------------------------------
#
# We use Essentia's MAEST model (discogs-maest-10s-pw-2, v2 — trained on 3.3M
# full tracks) for the "similar songs" embedding instead of a hand-rolled
# MFCC/spectral-contrast pipeline. The model is loudness-robust by design and
# does its own feature extraction (log-mel spectrogram), so the old
# MusicExtractor profile / EQ-loudness preprocessing is not needed.
#
# The .pb graph is baked into the Docker image at build time (see Dockerfile)
# and its location is overridable via MAEST_MODEL_PATH.
#
# Official reference: https://github.com/MTG/essentia
#   src/examples/python/models/scripts/feature-extractors/maest/
#   discogs-maest-10s-pw-2_embeddings.py
DEFAULT_MAEST_MODEL_PATH = "/models/discogs-maest-10s-pw-2.pb"
# MAEST v2 exposes per-token 768-dim embeddings from this output tensor;
# full-track embedding = mean over tokens, then L2-normalized (see l2_normalize).
MAEST_EMBEDDING_OUTPUT = "PartitionedCall/Identity_7"
# The model expects 16 kHz mono input (official example uses
# sampleRate=16000, resampleQuality=4).
MAEST_SAMPLE_RATE = 16000

# The ~344 MB graph is loaded lazily on the first request (and only once per
# process); loading it inside the request path per call would dominate
# latency. A lock keeps concurrent first requests from each building a graph.
_predictor = None
_predictor_lock = threading.Lock()


def _get_predictor():
    """Lazily build and cache the MAEST predictor (double-checked locking)."""
    global _predictor
    if _predictor is None:
        with _predictor_lock:
            if _predictor is None:
                _predictor = es.TensorflowPredictMAEST(
                    graphFilename=os.environ.get("MAEST_MODEL_PATH", DEFAULT_MAEST_MODEL_PATH),
                    output=MAEST_EMBEDDING_OUTPUT,
                )
    return _predictor


def l2_normalize(v):
    """Return v on the unit L2 sphere. A zero vector stays zero (no NaN)."""
    v = np.asarray(v, dtype=np.float64)
    norm = np.linalg.norm(v)
    if norm == 0:
        return np.zeros_like(v)
    return v / norm


def make_embedding(tmp_path):
    """Run the MAEST model over the audio in tmp_path.

    Returns the full-track embedding: the per-token 768-dim vectors from the
    model are averaged (one vector per ~10s window) and L2-normalized, so
    the result is a single 768-dim unit vector — the form stored in
    song_embeddings (VECTOR(768), cosine distance).
    """
    audio = es.MonoLoader(
        filename=tmp_path,
        sampleRate=MAEST_SAMPLE_RATE,
        resampleQuality=4,
    )()

    # (1, 1, n_tokens, 768) — one embedding per ~10s window; drop the leading
    # batch dims.
    tokens = np.asarray(_get_predictor()(audio), dtype=np.float64)
    if tokens.ndim != 4 or tokens.shape[-1] != 768:
        raise ValueError(
            f"unexpected MAEST output shape {tokens.shape}; expected (1, 1, n_tokens, 768)"
        )
    tokens = tokens[0, 0]
    if tokens.shape[0] == 0:
        # No 10s window could be produced (pathological input); fail loudly
        # instead of storing a NaN row (mean over an empty axis is NaN).
        raise ValueError("MAEST produced no token embeddings")
    track_embedding = tokens.mean(axis=0)
    return l2_normalize(track_embedding)


@app.post("/process-audio/")
def process_audio(file: UploadFile = File(...)):  # noqa: B008 — FastAPI dependency, standard pattern
    """Sync handler on purpose: MAEST inference is CPU-bound and blocks while
    it runs, so FastAPI runs it in the threadpool instead of stalling the
    event loop. (Concurrency model: one in-flight request per worker; the
    model graph is shared read-only — Essentia's thread-safety caveat is
    about concurrent calls on the *same audio buffer*. See
    .github/instructions/audio-processor.instructions.md.)"""
    # Reject non-audio uploads early.
    if not (file.content_type.startswith("audio/") or file.content_type == "application/octet-stream"):
        raise HTTPException(status_code=400, detail=f"Unsupported file type: {file.content_type}")

    # Use a unique temp file so concurrent requests don't clobber each other.
    fd, tmp_path = tempfile.mkstemp(suffix=".audio")
    try:
        with os.fdopen(fd, "wb") as f:
            f.write(file.file.read())

        embedding = make_embedding(tmp_path)

        return JSONResponse(content={"embedding": embedding.tolist()})

    except HTTPException:
        raise
    except Exception:
        # Log internals server-side; the client gets a sanitized message
        # (no native-library error text or paths in the response body).
        logger.exception("audio processing failed")
        raise HTTPException(status_code=500, detail="Audio processing failed")
    finally:
        try:
            os.remove(tmp_path)
        except OSError:
            pass


@app.get("/")
def read_root():
    return {"message": "Audio processing service is running"}


def main():
    """Entry point for running the audio processor service."""
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)


if __name__ == "__main__":
    main()
