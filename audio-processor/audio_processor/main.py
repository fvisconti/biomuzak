from fastapi import FastAPI, File, UploadFile, HTTPException
from fastapi.responses import JSONResponse
import numpy as np
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
    if not file.content_type.startswith("audio/"):
        raise HTTPException(status_code=400, detail="Unsupported file type")

    try:
        # Read audio file content
        contents = await file.read()

        # To process the audio file in memory, we need to write it to a temporary file,
        # as Essentia's standard loaders work with file paths.
        with open("temp_audio_file", "wb") as f:
            f.write(contents)

        # Load audio into Essentia
        loader = es.MonoLoader(filename="temp_audio_file")
        audio = loader()

        # Define the features to extract
        features_extractor = es.Extractor(
            lowlevel_spectral_contrast=True,
            lowlevel_mfcc=True,
            rhythm=True
        )

        # Extract features
        features, features_frames = features_extractor(audio)

        # Aggregate features
        mfcc_mean = np.mean(features_frames['lowlevel.mfcc'], axis=0)
        mfcc_std = np.std(features_frames['lowlevel.mfcc'], axis=0)
        scontrast_mean = np.mean(features_frames['lowlevel.spectral_contrast'], axis=0)
        scontrast_std = np.std(features_frames['lowlevel.spectral_contrast'], axis=0)

        # Normalize each feature component to a 0-1 range
        mfcc_mean_norm = normalize(mfcc_mean)
        mfcc_std_norm = normalize(mfcc_std)
        scontrast_mean_norm = normalize(scontrast_mean)
        scontrast_std_norm = normalize(scontrast_std)

        # Concatenate all normalized features into a single vector (38 dimensions)
        embedding = np.concatenate([
            mfcc_mean_norm,
            mfcc_std_norm,
            scontrast_mean_norm,
            scontrast_std_norm,
        ]).tolist()

        return JSONResponse(content={"embedding": embedding})

    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/")
def read_root():
    return {"message": "Audio processing service is running"}

def main():
    """Entry point for running the audio processor service."""
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)

if __name__ == "__main__":
    main()
