import os
import tempfile

import numpy as np
from fastapi import FastAPI, File, UploadFile, HTTPException
from fastapi.responses import JSONResponse
import essentia.standard as es

app = FastAPI()


def normalize(v):
    """Normalize a vector to the 0-1 range."""
    # A check to avoid division by zero if max == min
    if (np.max(v) - np.min(v)) == 0:
        return v - np.min(v)
    return (v - np.min(v)) / (np.max(v) - np.min(v))


@app.post("/process-audio/")
async def process_audio(file: UploadFile = File(...)):
    # Reject non-audio uploads early.
    if not (file.content_type.startswith("audio/") or file.content_type == "application/octet-stream"):
        raise HTTPException(status_code=400, detail=f"Unsupported file type: {file.content_type}")

    # Use a unique temp file so concurrent requests don't clobber each other.
    fd, tmp_path = tempfile.mkstemp(suffix=".audio")
    try:
        with os.fdopen(fd, "wb") as f:
            f.write(await file.read())

        # Load audio into Essentia.
        loader = es.MonoLoader(filename=tmp_path)
        audio = loader()

        # Use the high-level MusicExtractor to obtain low-level features.
        extractor = es.Extractor()
        _features, features_frames = extractor(audio)

        mfcc = np.asarray(features_frames["lowlevel.mfcc"])                    # (frames, 13)
        spectral_contrast = np.asarray(features_frames["lowlevel.spectral_contrast"])  # (frames, 6)

        mfcc_mean = np.mean(mfcc, axis=0)
        mfcc_std = np.std(mfcc, axis=0)
        sc_mean = np.mean(spectral_contrast, axis=0)
        sc_std = np.std(spectral_contrast, axis=0)

        # 13 + 13 + 6 + 6 = 38 dimensional embedding.
        embedding = np.concatenate([
            normalize(mfcc_mean).flatten(),
            normalize(mfcc_std).flatten(),
            normalize(sc_mean).flatten(),
            normalize(sc_std).flatten(),
        ]).tolist()

        return JSONResponse(content={"embedding": embedding})

    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
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
