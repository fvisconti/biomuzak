import pytest
import numpy as np
from unittest.mock import Mock, patch, AsyncMock
from fastapi.testclient import TestClient
from fastapi import UploadFile
import io

from audio_processor.main import app, normalize, process_audio

client = TestClient(app)


class TestNormalize:
    def test_normalize_regular_vector(self):
        """Test normalization of a regular vector"""
        v = np.array([1.0, 2.0, 3.0, 4.0, 5.0])
        result = normalize(v)
        
        # Check that result is between 0 and 1
        assert result.min() >= 0.0
        assert result.max() <= 1.0
        # Check that min maps to 0 and max maps to 1
        assert result.min() == 0.0
        assert result.max() == 1.0
        
    def test_normalize_constant_vector(self):
        """Test normalization of a constant vector (all same values)"""
        v = np.array([5.0, 5.0, 5.0, 5.0])
        result = normalize(v)
        
        # All values should be 0 after normalization
        assert np.all(result == 0.0)
        
    def test_normalize_negative_values(self):
        """Test normalization with negative values"""
        v = np.array([-2.0, -1.0, 0.0, 1.0, 2.0])
        result = normalize(v)
        
        assert result.min() >= 0.0
        assert result.max() <= 1.0
        

class TestProcessAudioEndpoint:
    def test_root_endpoint(self):
        """Test the root endpoint returns expected message"""
        response = client.get("/")
        assert response.status_code == 200
        assert response.json() == {"message": "Audio processing service is running"}
        
    @patch('audio_processor.main.es.MonoLoader')
    @patch('audio_processor.main.es.extractor.Extractor')
    def test_process_audio_success(self, mock_extractor_class, mock_loader_class):
        """Test successful audio processing"""
        # Mock the audio loader
        mock_loader = Mock()
        mock_audio = np.random.rand(44100)  # 1 second of audio at 44.1kHz
        mock_loader.return_value = mock_audio
        mock_loader_class.return_value = mock_loader
        
        # Mock the feature extractor
        mock_extractor = Mock()
        mock_features = {}
        mock_features_frames = {
            'lowlevel.mfcc': np.random.rand(100, 13),  # 13 MFCC coefficients
            'lowlevel.spectral_contrast': np.random.rand(100, 6),  # 6 spectral contrast bands
        }
        mock_extractor.return_value = (mock_features, mock_features_frames)
        mock_extractor_class.return_value = mock_extractor
        
        # Create a mock audio file
        audio_content = b"mock audio content"
        files = {"file": ("test.mp3", io.BytesIO(audio_content), "audio/mpeg")}
        
        response = client.post("/process-audio/", files=files)
        
        assert response.status_code == 200
        data = response.json()
        assert "embedding" in data
        assert isinstance(data["embedding"], list)
        # Embedding should be 38 dimensional: 13 + 13 + 6 + 6
        assert len(data["embedding"]) == 38
        
    def test_process_audio_invalid_file_type(self):
        """Test processing with non-audio file"""
        # Create a mock non-audio file
        files = {"file": ("test.txt", io.BytesIO(b"not audio"), "text/plain")}
        
        response = client.post("/process-audio/", files=files)
        
        assert response.status_code == 400
        assert "Unsupported file type" in response.json()["detail"]
        
    def test_process_audio_missing_file(self):
        """Test processing without file"""
        response = client.post("/process-audio/")
        
        assert response.status_code == 422  # Unprocessable Entity
        
    @patch('audio_processor.main.es.MonoLoader')
    def test_process_audio_processing_error(self, mock_loader_class):
        """Test handling of processing errors"""
        # Mock the loader to raise an exception
        mock_loader = Mock()
        mock_loader.side_effect = Exception("Audio processing failed")
        mock_loader_class.return_value = mock_loader
        
        # Create a mock audio file
        files = {"file": ("test.mp3", io.BytesIO(b"mock audio"), "audio/mpeg")}
        
        response = client.post("/process-audio/", files=files)
        
        assert response.status_code == 500
        assert "Audio processing failed" in response.json()["detail"]


class TestEmbeddingDimensions:
    @patch('audio_processor.main.es.MonoLoader')
    @patch('audio_processor.main.es.extractor.Extractor')
    def test_embedding_dimensions(self, mock_extractor_class, mock_loader_class):
        """Test that the embedding has exactly 38 dimensions"""
        # Mock the audio loader
        mock_loader = Mock()
        mock_audio = np.random.rand(44100)
        mock_loader.return_value = mock_audio
        mock_loader_class.return_value = mock_loader
        
        # Mock the feature extractor with specific dimensions
        mock_extractor = Mock()
        mock_features = {}
        mock_features_frames = {
            'lowlevel.mfcc': np.random.rand(100, 13),  # 13 MFCC
            'lowlevel.spectral_contrast': np.random.rand(100, 6),  # 6 spectral contrast
        }
        mock_extractor.return_value = (mock_features, mock_features_frames)
        mock_extractor_class.return_value = mock_extractor
        
        files = {"file": ("test.mp3", io.BytesIO(b"mock audio"), "audio/mpeg")}
        response = client.post("/process-audio/", files=files)
        
        assert response.status_code == 200
        embedding = response.json()["embedding"]
        
        # 13 (mfcc_mean) + 13 (mfcc_std) + 6 (scontrast_mean) + 6 (scontrast_std) = 38
        assert len(embedding) == 38
        
        # All values should be between 0 and 1 (normalized)
        assert all(0.0 <= val <= 1.0 for val in embedding)


if __name__ == "__main__":
    pytest.main([__file__, "-v"])