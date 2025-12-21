import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import Login from './pages/Login';
import Register from './pages/Register';
import Upload from './pages/Upload';
import Playlists from './pages/Playlists';
import PlaylistDetails from './pages/PlaylistDetails';
import Library from './pages/Library';
import MainLayout from './components/Layout/MainLayout';
import { AuthProvider, useAuth } from './context/AuthContext';
import { PlayerProvider } from './context/PlayerContext';
import { DragProvider } from './context/DragContext';
import { PlaylistProvider } from './context/PlaylistContext';
import PlayerBar from './components/Layout/PlayerBar';

// Protected Route Component
const ProtectedRoute = ({ children }) => {
  const { token, loading } = useAuth();

  if (loading) {
    return null;
  }

  if (!token) {
    return <Navigate to="/login" replace />;
  }

  return children;
};

const App = () => {
  return (
    <AuthProvider>
      <PlaylistProvider>
        <PlayerProvider>
          <DragProvider>
            <Router>
              <Routes>
                <Route path="/login" element={<Login />} />
                <Route path="/register" element={<Register />} />

                <Route path="/" element={
                  <ProtectedRoute>
                    <MainLayout>
                      <Navigate to="/library" replace />
                    </MainLayout>
                  </ProtectedRoute>
                } />

                <Route path="/upload" element={
                  <ProtectedRoute>
                    <MainLayout>
                      <Upload />
                    </MainLayout>
                  </ProtectedRoute>
                } />

                <Route path="/playlists" element={
                  <ProtectedRoute>
                    <MainLayout>
                      <Playlists />
                    </MainLayout>
                  </ProtectedRoute>
                } />

                <Route path="/playlists/:playlistID" element={
                  <ProtectedRoute>
                    <MainLayout>
                      <PlaylistDetails />
                    </MainLayout>
                  </ProtectedRoute>
                } />

                <Route path="/library" element={
                  <ProtectedRoute>
                    <MainLayout>
                      <Library />
                    </MainLayout>
                  </ProtectedRoute>
                } />
              </Routes>
              <PlayerBar />
            </Router>
          </DragProvider>
        </PlayerProvider>
      </PlaylistProvider>
    </AuthProvider>
  );
};

export default App;
