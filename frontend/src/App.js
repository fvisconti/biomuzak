import React from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import Login from './components/Login';
import Register from './components/Register';
import Library from './components/Library';
import Upload from './components/Upload';
import Playlists from './components/Playlists';
import PrivateRoute from './components/PrivateRoute';
import './App.css';
import AdminUsers from './components/AdminUsers';

function App() {
  return (
    <Router>
      <div className="App">
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/register" element={<Register />} />
          <Route 
            path="/library" 
            element={
              <PrivateRoute>
                <Library />
              </PrivateRoute>
            } 
          />
          <Route 
            path="/upload" 
            element={
              <PrivateRoute>
                <Upload />
              </PrivateRoute>
            } 
          />
          <Route 
            path="/playlists" 
            element={
              <PrivateRoute>
                <Playlists />
              </PrivateRoute>
            } 
          />
          <Route 
            path="/admin" 
            element={
              <PrivateRoute>
                <AdminUsers />
              </PrivateRoute>
            } 
          />
          <Route path="/" element={<Navigate to="/library" replace />} />
        </Routes>
      </div>
    </Router>
  );
}

export default App;
