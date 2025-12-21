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
    # Relaxed check to support various browser upload types
    if not (file.content_type.startswith("audio/") or file.content_type == "application/octet-stream"):
        pass # Allow it to proceed and let Essentia fail if it's invalid, or log warning
        # raise HTTPException(status_code=400, detail=f"Unsupported file type: {file.content_type}")

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

        # Use specific extractors to avoid parameter naming issues
        # MFCC
        w_hann = es.Windowing(type='hann')
        spectrum = es.Spectrum()
        mfcc_algo = es.MFCC()
        
        mfccs = []
        # Frame-wise processing usually required, but let's see if we can use on whole audio?
        # es.MFCC expects spectrum.
        # Let's use the MusicExtractor for high-level if available, OR simple frame loop.
        
        # Simpler approach: FrameGenerator
        for frame in es.FrameGenerator(audio, frameSize=1024, hopSize=512, startFromZero=True):
            spec = spectrum(w_hann(frame))
            mfcc_bands, mfcc_coeffs = mfcc_algo(spec)
            mfccs.append(mfcc_coeffs)
            
        mfccs = np.array(mfccs)
        mfcc_mean = np.mean(mfccs, axis=0)
        mfcc_std = np.std(mfccs, axis=0)

        # Spectral Contrast
        sc_algo = es.SpectralContrast(frameSize=1024)
        scs = []
        for frame in es.FrameGenerator(audio, frameSize=1024, hopSize=512, startFromZero=True):
             spec = spectrum(w_hann(frame))
             sc_val, sc_valley = sc_algo(spec)
             scs.append(sc_val)
             
        scs = np.array(scs)
        scontrast_mean = np.mean(scs, axis=0)
        scontrast_std = np.std(scs, axis=0)

        # Normalize
        mfcc_mean_norm = normalize(mfcc_mean)
        mfcc_std_norm = normalize(mfcc_std)
        scontrast_mean_norm = normalize(scontrast_mean)
        scontrast_std_norm = normalize(scontrast_std)

        # Flatten and concatenate
        embedding = np.concatenate([
            mfcc_mean_norm.flatten(),
            mfcc_std_norm.flatten(),
            scontrast_mean_norm.flatten(),
            scontrast_std_norm.flatten(),
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
