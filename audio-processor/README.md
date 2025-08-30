# Audio Processing Microservice

This microservice is a Python-based web service that generates feature embeddings from audio files using the Essentia library. It's designed to be a component in a larger system for tasks like finding similar songs.

## Requirements

- Docker

## Getting Started

To get the service running, you'll need to build and run the Docker container. This project uses `pyproject.toml` for dependency management.

1.  **Build the Docker image:**

    ```bash
    docker build -t audio-processor .
    ```

2.  **Run the Docker container:**

    ```bash
    docker run -p 8000:8000 audio-processor
    ```

The service will be available at `http://localhost:8000`.

## API Usage

### Process an Audio File

- **Endpoint:** `/process-audio/`
- **Method:** `POST`
- **Request Body:** `multipart/form-data` with a single field named `file` containing the audio file.

**Example using `curl`:**

```bash
curl -X POST -F "file=@/path/to/your/song.mp3" http://localhost:8000/process-audio/
```

**Successful Response:**

- **Status Code:** `200 OK`
- **Body:** A JSON object containing a 38-dimensional feature embedding vector. The vector is normalized to a 0-1 range.

```json
{
  "embedding": [
    0.87,
    0.12,
    // ... 36 more float values
  ]
}
```

### Error Responses

- **Status Code:** `400 Bad Request` - If the uploaded file is not an audio file.
- **Status Code:** `500 Internal Server Error` - If there was an error processing the audio file.
