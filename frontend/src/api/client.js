import axios from 'axios';

// Create axios instance with base configuration
const apiClient = axios.create({
    baseURL: '/api',
    timeout: 30000,
    headers: {
        'Content-Type': 'application/json',
    },
});

// Request interceptor - add auth token to all requests
apiClient.interceptors.request.use(
    (config) => {
        const token = localStorage.getItem('token');
        if (token) {
            config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
    },
    (error) => {
        console.error('Request error:', error);
        return Promise.reject(error);
    }
);

// Response interceptor - standardize error handling
apiClient.interceptors.response.use(
    (response) => response,
    (error) => {
        // Handle 401 - unauthorized (invalid/expired token)
        if (error.response?.status === 401) {
            localStorage.removeItem('token');
            // Only redirect if not already on login/register page
            if (!['/login', '/register'].includes(window.location.pathname)) {
                window.location.href = '/login';
            }
        }

        // Extract error message from various possible formats
        let errorMessage = 'An error occurred';
        
        if (error.response?.data) {
            const data = error.response.data;
            // Check for standardized error format
            if (data.message) {
                errorMessage = data.message;
            } else if (data.error) {
                errorMessage = data.error;
            } else if (typeof data === 'string') {
                errorMessage = data;
            }
        } else if (error.message) {
            errorMessage = error.message;
        }

        // Create a standard error object
        const standardError = new Error(errorMessage);
        standardError.response = error.response;
        standardError.status = error.response?.status;
        
        console.error('API Error:', {
            message: errorMessage,
            status: error.response?.status,
            url: error.config?.url,
        });

        return Promise.reject(standardError);
    }
);

export default apiClient;
