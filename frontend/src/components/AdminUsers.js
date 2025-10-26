import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { getMe, adminCreateUser } from '../api';
import './Auth.css';

function AdminUsers() {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [isAdmin, setIsAdmin] = useState(false);
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    let mounted = true;
    getMe()
      .then(({ data }) => {
        if (mounted) {
          setIsAdmin(!!data?.is_admin);
        }
      })
      .catch(() => {
        setIsAdmin(false);
      })
      .finally(() => setLoading(false));
    return () => { mounted = false; };
  }, []);

  useEffect(() => {
    if (!loading && !isAdmin) {
      navigate('/library', { replace: true });
    }
  }, [loading, isAdmin, navigate]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setMessage('');
    try {
      await adminCreateUser(username, password);
      setMessage(`User '${username}' created`);
      setUsername('');
      setPassword('');
    } catch (err) {
      const msg = err?.response?.data || 'Failed to create user';
      setError(typeof msg === 'string' ? msg : 'Failed to create user');
    }
  };

  if (loading) return <div className="auth-container"><p>Loading…</p></div>;

  return (
    <div className="auth-container">
      <h2>Admin: Create User</h2>
      {message && <div className="success-message">{message}</div>}
      {error && <div className="error-message">{error}</div>}
      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label htmlFor="username">Username</label>
          <input
            id="username"
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            required
          />
        </div>
        <div className="form-group">
          <label htmlFor="password">Password</label>
          <input
            id="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </div>
        <button type="submit" className="auth-button">Create</button>
      </form>
    </div>
  );
}

export default AdminUsers;
