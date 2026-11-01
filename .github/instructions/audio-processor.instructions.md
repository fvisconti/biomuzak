---
description: 'Essentia / FastAPI / numpy rules for the audio-processor microservice'
applyTo: 'audio-processor/**/*.py,audio-processor/pyproject.toml,audio-processor/Dockerfile'
---

# Audio Processor Instructions

Applies to all Python code in `audio-processor/`. The service is a FastAPI
microservice that extracts 38-dim audio feature embeddings using Essentia and
returns them as JSON for the Go backend to store in Postgres.

## Architecture

- **One endpoint, one job.** `POST /process-audio/` accepts a single audio file
  (multipart/form-data, field `file`), extracts features, and returns
  `{"embedding": [38 floats]}`. Do not add business logic, persistence, or
  multi-file batch processing here — that belongs in the Go backend.
- **The 38-dim contract is fixed.** The embedding is
  `concat(normalize(mfcc_mean), normalize(mfcc_std), normalize(sc_mean),
  normalize(sc_std))` = 13 + 13 + 6 + 6 = 38. Changing the dimension or the
  feature set is a **breaking change** that requires a new migration (drop and
  recreate `song_embeddings`), a Go-side update in `pkg/db/similar.go`, and a
  `docs/CHANGELOG.md` entry marked **BREAKING**.
- **Normalization is min-max to [0, 1].** The `normalize()` helper guards
  against division by zero (constant vector → all zeros). Do not switch to
  z-score or L2 normalization without a recorded decision and a migration.

## Essentia usage

- Use `essentia.standard` (the high-level API). Do not drop to the low-level
  `essentia` C++ bindings.
- `es.MonoLoader` for loading; `es.Extractor` for feature extraction. The
  extractor returns `(features, features_frames)`; we use `features_frames`
  (per-frame) and aggregate with `np.mean` / `np.std` over the frame axis.
- Essentia is a heavy native dependency (FFmpeg, FFTW, Chromaprint). The
  Dockerfile installs system libs (`libfftw3-dev`, `libavcodec-dev`, etc.)
  before `poetry install`. Do not remove these without verifying the build.
- Essentia is **not thread-safe** for concurrent `Extractor` calls on the same
  audio buffer. The current single-request-at-a-time model is fine; if
  concurrency is added, create a fresh `Extractor` per request.

## FastAPI rules

- Validate input early: reject non-audio `content_type` with 400 before
  touching disk.
- Use `tempfile.mkstemp` for the temp file; always clean up in `finally`.
- Return `JSONResponse` (not a bare dict) so the response is explicit.
- Catch `HTTPException` separately from generic `Exception` so 400s are not
  swallowed into 500s.
- The service is stateless; no in-memory cache, no session, no DB.

## Testing

- Tests live in `audio-processor/tests/` (package) and `audio-processor/test_main.py`.
- **Never load real audio in unit tests.** Mock `es.MonoLoader` and
  `es.Extractor` with `unittest.mock.patch`. The test fixtures are synthetic
  numpy arrays shaped like real Essentia output (`(frames, 13)` for MFCC,
  `(frames, 6)` for spectral contrast).
- Use `fastapi.testclient.TestClient` for endpoint tests.
- Assert on the **shape and range** of the embedding (38 floats, each in
  [0, 1]), not on exact values.
- Run with `pytest` from the `audio-processor/` directory.

## Dependencies

- Managed by `pyproject.toml` (Poetry). Python `>=3.11,<3.12`.
- Do not add a dependency without a recorded reason. The current set is
  minimal: `fastapi`, `uvicorn`, `python-multipart`, `numpy`, `essentia`.
- Dev deps: `pytest`, `httpx`, `pytest-asyncio`, `ruff`.
- Lint with `ruff` (configured in `pyproject.toml`).

## Docker

- Base image: `python:3.11-slim`.
- System libs are installed via `apt-get` before `poetry install`.
- The service runs as `uvicorn` on port 8000.
- Health check: `GET /` returns `{"message": "Audio processing service is
  running"}`.
