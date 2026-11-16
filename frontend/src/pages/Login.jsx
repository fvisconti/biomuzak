import React, { useState } from 'react';
import {
    Box,
    Button,
    FormControl,
    FormLabel,
    Input,
    VStack,
    Heading,
    Text,
    useToast,
    Link,
    Container,
} from '@chakra-ui/react';
import { useAuth } from '../context/AuthContext';
import { useNavigate, Link as RouterLink } from 'react-router-dom';

const Login = () => {
    const [username, setUsername] = useState('');
    const [password, setPassword] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const { login, logout } = useAuth();
    const navigate = useNavigate();
    const toast = useToast();

    const handleSubmit = async (e) => {
        e.preventDefault();
        setIsLoading(true);
            const result = await login(username, password);
        setIsLoading(false);

        if (result.success) {
            navigate('/');
        } else {
            toast({
                title: 'Login Failed',
                description: result.message,
                status: 'error',
                duration: 3000,
                isClosable: true,
            });
        }
    };

    return (
        <Box bg="gray.900" h="100vh" display="flex" alignItems="center">
            <Container maxW="md">
                <Box bg="gray.800" p={8} borderRadius="lg" boxShadow="lg" border="1px solid" borderColor="gray.700">
                    <VStack spacing={4} as="form" onSubmit={handleSubmit}>
                        <Heading size="lg" color="white">BioMuzak</Heading>
                        <Text color="gray.400">Sign in to your account</Text>

                        <FormControl id="username" isRequired>
                            <FormLabel color="gray.300">Username</FormLabel>
                            <Input
                                type="text"
                                value={username}
                                onChange={(e) => setUsername(e.target.value)}
                                bg="gray.700"
                                border="none"
                                color="white"
                                _focus={{ ring: 2, ringColor: 'blue.500' }}
                            />
                        </FormControl>

                        <FormControl id="password" isRequired>
                            <FormLabel color="gray.300">Password</FormLabel>
                            <Input
                                type="password"
                                value={password}
                                onChange={(e) => setPassword(e.target.value)}
                                bg="gray.700"
                                border="none"
                                color="white"
                                _focus={{ ring: 2, ringColor: 'blue.500' }}
                            />
                        </FormControl>

                        <Button
                            type="submit"
                            colorScheme="blue"
                            w="100%"
                            isLoading={isLoading}
                            mt={4}
                        >
                            Sign In
                        </Button>

                        <Text fontSize="sm" color="gray.400">
                            Don't have an account?{' '}
                            <Link as={RouterLink} to="/register" color="blue.400">
                                Register
                            </Link>
                        </Text>
                        <Button
                            variant="outline"
                            colorScheme="red"
                            size="sm"
                            onClick={() => {
                                logout();
                                setUsername('');
                                setPassword('');
                            }}
                        >
                            Clear cached session
                        </Button>
                    </VStack>
                </Box>
            </Container>
        </Box>
    );
};

export default Login;
