import io
import sys
import types
from unittest.mock import Mock, patch

import numpy as np
import pytest
from fastapi.testclient import TestClient

# The audio-processor is normally run inside its Docker image, where Essentia
# is available. The local test environment (no Docker) does not have Essentia,
# so we stub out the module before importing the application. These tests
# verify the MAEST pipeline wiring (averaging, L2 normalization, model
# parameters) with the Essentia algorithms mocked.
essentia_mod = types.ModuleType("essentia")
essentia_mod.__path__ = []  # mark as a package so `import essentia.standard` resolves
sys.modules["essentia"] = essentia_mod
standard_mod = types.ModuleType("essentia.standard")
# Attributes patched in the tests below resolve through this module.
standard_mod.MonoLoader = object
standard_mod.TensorflowPredictMAEST = object
sys.modules["essentia.standard"] = standard_mod

import audio_processor.main as appmod
from audio_processor.main import app, l2_normalize

client = TestClient(app)


@pytest.fixture(autouse=True)
def _reset_cached_predictor():
    """The predictor is built once per process; reset between tests."""
    appmod._predictor = None
    yield
    appmod._predictor = None

# The 768-dim MAEST contract shared with the Go side (pkg/db/similar.go)
# and the VECTOR(768) column (migration 0010).
EMBEDDING_DIM = 768
# Official model (discogs-maest-10s-pw-2): token embeddings come out of
# PartitionedCall/Identity_7 with shape (1, n_tokens, 768).
EMBEDDING_OUTPUT = "PartitionedCall/Identity_7"


def make_files(content_type="audio/mpeg"):
    return {"file": ("test.mp3", io.BytesIO(b"mock audio"), content_type)}


def post(files):
    return client.post("/process-audio/", files=files)


class TestL2Normalize:
    def test_unit_norm(self):
        rng = np.random.default_rng(42)
        v = rng.standard_normal(768)
        result = l2_normalize(v)
        assert abs(np.linalg.norm(result) - 1.0) < 1e-9

    def test_zero_vector_stays_zero(self):
        """A zero vector must normalize to zero, not NaN (no division by 0)."""
        result = l2_normalize(np.zeros(768))
        assert np.all(np.isfinite(result))
        assert np.all(result == 0.0)

    def test_preserves_direction(self):
        v = np.zeros(768)
        v[0] = 3.0
        v[1] = 4.0
        result = l2_normalize(v)
        assert np.allclose(result, [3.0 / 5.0, 4.0 / 5.0] + [0.0] * 766)


class TestProcessAudioEndpoint:
    def test_root_endpoint(self):
        response = client.get("/")
        assert response.status_code == 200
        assert response.json() == {"message": "Audio processing service is running"}

    @patch("audio_processor.main.es.TensorflowPredictMAEST")
    @patch("audio_processor.main.es.MonoLoader")
    def test_process_audio_success(self, mock_loader_class, mock_model_class):
        mock_loader = Mock()
        mock_loader.return_value = np.random.rand(160000)  # 10s @ 16kHz
        mock_loader_class.return_value = mock_loader

        rng = np.random.default_rng(7)
        # MAEST emits (1, n_tokens, 768); a real batch dimension of 1 is what
        # TensorflowPredictMAEST returns for a single track.
        tokens = rng.standard_normal((1, 1, 4, 768))
        mock_model = Mock(return_value=tokens)
        mock_model_class.return_value = mock_model

        response = post(make_files())

        assert response.status_code == 200
        embedding = response.json()["embedding"]
        assert len(embedding) == EMBEDDING_DIM
        # The embedding must be L2-normalized (unit norm).
        assert abs(np.linalg.norm(embedding) - 1.0) < 1e-6
        # The predictor is built once and reused across requests.
        assert mock_model_class.call_count == 1
        post(make_files())
        assert mock_model_class.call_count == 1

    @patch("audio_processor.main.es.TensorflowPredictMAEST")
    @patch("audio_processor.main.es.MonoLoader")
    def test_token_averaging(self, mock_loader_class, mock_model_class):
        """Per-token embeddings are averaged before normalization."""
        mock_loader_class.return_value.return_value = np.random.rand(160000)

        tokens = np.zeros((1, 1, 3, 768))
        tokens[0, 0, :, 0] = 1.0  # three identical tokens along dim 0
        mock_model = Mock(return_value=tokens)
        mock_model_class.return_value = mock_model

        response = post(make_files())
        assert response.status_code == 200
        embedding = np.asarray(response.json()["embedding"])
        # Mean over tokens is (1, 0, ..., 0); L2-norm of that is itself.
        assert np.allclose(embedding, [1.0] + [0.0] * 767)

    @patch("audio_processor.main.es.TensorflowPredictMAEST")
    @patch("audio_processor.main.es.MonoLoader")
    def test_loader_uses_model_sample_rate(self, mock_loader_class, mock_model_class):
        """MAEST expects 16 kHz input; the loader must resample to it."""
        mock_model_class.return_value.return_value = np.zeros((1, 1, 1, 768))

        response = post(make_files())
        assert response.status_code == 200

        kwargs = mock_loader_class.call_args.kwargs
        assert kwargs["sampleRate"] == 16000
        assert "resampleQuality" in kwargs

    @patch("audio_processor.main.es.TensorflowPredictMAEST")
    @patch("audio_processor.main.es.MonoLoader")
    def test_model_output_and_env_path(self, mock_loader_class, mock_model_class, monkeypatch):
        """The model is constructed with the documented embedding output and
        the MAEST_MODEL_PATH (env-overridable)."""
        monkeypatch.setenv("MAEST_MODEL_PATH", "/models/custom.pb")
        mock_model = Mock(return_value=np.ones((1, 1, 1, 768)))
        mock_model_class.return_value = mock_model
        mock_loader_class.return_value.return_value = np.random.rand(160000)

        response = post(make_files())
        assert response.status_code == 200

        kwargs = mock_model_class.call_args.kwargs
        assert kwargs["graphFilename"] == "/models/custom.pb"
        assert kwargs["output"] == EMBEDDING_OUTPUT

    def test_process_audio_invalid_file_type(self):
        response = post(make_files(content_type="text/plain"))
        assert response.status_code == 400
        assert "Unsupported file type" in response.json()["detail"]

    def test_process_audio_missing_file(self):
        response = client.post("/process-audio/")
        assert response.status_code == 422

    @patch("audio_processor.main.es.MonoLoader")
    def test_process_audio_processing_error(self, mock_loader_class):
        mock_loader = Mock()
        mock_loader.side_effect = Exception("mock failure reason /secret/path/x.pb")
        mock_loader_class.return_value = mock_loader

        response = post(make_files())

        assert response.status_code == 500
        # Sanitized message: internal error text never reaches the client.
        assert response.json()["detail"] == "Audio processing failed"
        assert "mock failure reason" not in response.json()["detail"]


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
