---
description: 'Essentia / FastAPI / numpy rules for the audio-processor microservice'
applyTo: 'audio-processor/**/*.py,audio-processor/pyproject.toml,audio-processor/Dockerfile'
---

# Audio Processor Instructions

Applies to all Python code in `audio-processor/`. The service is a FastAPI
microservice that generates **768-dim MAEST embeddings** (Essentia's
`discogs-maest-10s-pw-2` model, v2) and returns them as JSON for the Go
backend to store in Postgres.

## Architecture

- **One endpoint, one job.** `POST /process-audio/` accepts a single audio
  file (multipart/form-data, field `file`), extracts the embedding, and
  returns `{"embedding": [768 floats]}`. Do not add business logic,
  persistence, or multi-file batch processing here — that belongs in the Go
  backend.
- **The 768-dim MAEST contract is fixed.** The embedding is the mean of the
  model's per-10s-window token vectors (`PartitionedCall/Identity_7`, shape
  `(1, n_tokens, 768)`), then L2-normalized to unit norm. Changing the model,
  dimension, or the mean+L2 pipeline is a **breaking change** that requires a
  new migration (drop and recreate `song_embeddings`), a Go-side update in
  `pkg/db/similar.go`, and a `docs/CHANGELOG.md` entry marked **BREAKING**.
- **Normalization is L2 to unit norm.** `l2_normalize()` guards against
  division by zero (zero vector → zeros, never NaN). The old min-max [0,1]
  normalization (pre-MAEST, 38-dim era) is gone; do not resurrect it.

## Essentia usage

- Use `essentia.standard` (the high-level API). Do not drop to the low-level
  `essentia` C++ bindings.
- `es.MonoLoader(filename=..., sampleRate=16000, resampleQuality=4)` for
  loading — MAEST expects 16 kHz mono. `es.TensorflowPredictMAEST`
  (`graphFilename`, `output="PartitionedCall/Identity_7")` for inference,
  exactly as the official example
  `src/examples/python/models/scripts/feature-extractors/maest/
  discogs-maest-10s-pw-2_embeddings.py`.
- The model graph (`.pb`, ~344 MB) is downloaded into the image at Docker
  build time; its location comes from `MAEST_MODEL_PATH` (default
  `/models/discogs-maest-10s-pw-2.pb`). Never download at request time.
- Essentia is a heavy native dependency (FFmpeg, FFTW, Chromaprint). The
  Dockerfile installs system libs (`libfftw3-dev`, `libavcodec-dev`, etc.)
  before `poetry install`. Do not remove these without verifying the build.
- Essentia is **not thread-safe** for concurrent predictor calls on the same
  audio buffer. The current single-request-at-a-time model is fine; if
  concurrency is added, serialize inference or build a fresh predictor per
  worker.

## FastAPI rules

- Validate input early: reject non-audio `content_type` with 400 before
  touching disk.
- Use `tempfile.mkstemp` for the temp file; always clean up in `finally`.
- Return `JSONResponse` (not a bare dict) so the response is explicit.
- Catch `HTTPException` separately from generic `Exception` so 400s are not
  swallowed into 500s.
- The service is stateless apart from the model path; no in-memory cache,
  no session, no DB.

## Testing

- Tests live in `audio-processor/tests/` (package) and
  `audio-processor/test_main.py`.
- **Never load real audio in unit tests.** Mock `es.MonoLoader` and
  `es.TensorflowPredictMAEST` with `unittest.mock.patch`. The test fixtures
  are synthetic numpy arrays shaped like real model output
  (`(1, n_tokens, 768)`).
- Use `fastapi.testclient.TestClient` for endpoint tests.
- Assert on the **shape and norm** of the embedding (768 floats, unit L2
  norm), not on exact values.
- Run with `pytest` from the `audio-processor/` directory.

## Dependencies

- Managed by `pyproject.toml` (Poetry). Python `>=3.11,<3.12`.
- Do not add a dependency without a recorded reason. The current set is
  minimal: `fastapi`, `uvicorn`, `python-multipart`, `numpy`, `essentia`.
- Dev deps: `pytest`, `httpx`, `pytest-asyncio`, `ruff`.
- Lint with `ruff` (configured in `pyproject.toml`).
