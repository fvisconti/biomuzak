import React, { createContext, useState, useContext, useEffect } from 'react';
import axios from 'axios';

const AuthContext = createContext(null);

export const AuthProvider = ({ children }) => {
    const [user, setUser] = useState(null);
    const [token, setToken] = useState(localStorage.getItem('token'));
    const [loading, setLoading] = useState(true);

    // Configure axios defaults
    if (token) {
        axios.defaults.headers.common['Authorization'] = `Bearer ${token}`;
    }

    useEffect(() => {
        const initAuth = async () => {
            if (token) {
                try {
                    const response = await axios.get('/api/auth/me');
                    if (response.data && response.data.username) {
                        setUser(response.data);
                    } else {
                        console.warn("Invalid user data received", response.data);
                        logout();
                    }
                } catch (error) {
                    console.error("Failed to fetch user", error);
                    logout();
                }
            }
            setLoading(false);
        };

        initAuth();
    }, [token]);

    const login = async (username, password) => {
        try {
            const response = await axios.post('/api/auth/login', { username, password });
            const newToken = response.data.token;

            localStorage.setItem('token', newToken);
            setToken(newToken);
            axios.defaults.headers.common['Authorization'] = `Bearer ${newToken}`;

            // Fetch user details immediately
            const userResponse = await axios.get('/api/auth/me');
            setUser(userResponse.data);

            return { success: true };
        } catch (error) {
            return {
                success: false,
                message: error.response?.data || "Login failed"
            };
        }
    };

    const register = async (username, email, password) => {
        try {
            await axios.post('/api/auth/register', { username, email, password });
            return { success: true };
        } catch (error) {
            return {
                success: false,
                message: error.response?.data || "Registration failed"
            };
        }
    };

    const logout = () => {
        localStorage.removeItem('token');
        setToken(null);
        setUser(null);
        delete axios.defaults.headers.common['Authorization'];
    };

    return (
        <AuthContext.Provider value={{ user, token, loading, login, register, logout }}>
            {children}
        </AuthContext.Provider>
    );
};

export const useAuth = () => useContext(AuthContext);
