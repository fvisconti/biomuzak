import React, { createContext, useState, useContext, useEffect, useCallback } from 'react';
import apiClient from '../api/client';

const AuthContext = createContext(null);

export const AuthProvider = ({ children }) => {
    const [user, setUser] = useState(null);
    const [token, setToken] = useState(localStorage.getItem('token'));
    const [loading, setLoading] = useState(true);

    const logout = useCallback(() => {
        localStorage.removeItem('token');
        setToken(null);
        setUser(null);
    }, []);

    useEffect(() => {
        const initAuth = async () => {
            if (token) {
                try {
                    const response = await apiClient.get('/auth/me');
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
    }, [token, logout]);

    const login = async (username, password) => {
        try {
            const response = await apiClient.post('/auth/login', { username, password });
            const newToken = response.data.token;

            localStorage.setItem('token', newToken);
            setToken(newToken);

            // Fetch user details immediately
            const userResponse = await apiClient.get('/auth/me');
            setUser(userResponse.data);

            return { success: true };
        } catch (error) {
            return {
                success: false,
                message: error.message || "Login failed"
            };
        }
    };

    const register = async (username, email, password) => {
        try {
            await apiClient.post('/auth/register', { username, email, password });
            return { success: true };
        } catch (error) {
            return {
                success: false,
                message: error.message || "Registration failed"
            };
        }
    };

    return (
        <AuthContext.Provider value={{ user, token, loading, login, register, logout }}>
            {children}
        </AuthContext.Provider>
    );
};

export const useAuth = () => useContext(AuthContext);
