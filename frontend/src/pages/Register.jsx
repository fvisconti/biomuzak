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

const Register = () => {
    const [username, setUsername] = useState('');
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const { register } = useAuth();
    const navigate = useNavigate();
    const toast = useToast();

    const handleSubmit = async (e) => {
        e.preventDefault();
        setIsLoading(true);
        const result = await register(username, email, password);
        setIsLoading(false);

        if (result.success) {
            toast({
                title: 'Account created.',
                description: "We've created your account for you. Please log in.",
                status: 'success',
                duration: 3000,
                isClosable: true,
            });
            navigate('/login');
        } else {
            toast({
                title: 'Registration Failed',
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
                        <Heading size="lg" color="white">Create Account</Heading>
                        <Text color="gray.400">Join BioMuzak today</Text>

                        <FormControl id="username" isRequired>
                            <FormLabel color="gray.300">Username</FormLabel>
                            <Input
                                type="text"
                                value={username}
                                onChange={(e) => setUsername(e.target.value)}
                                bg="gray.700"
                                border="none"
                                color="white"
                            />
                        </FormControl>

                        <FormControl id="email" isRequired>
                            <FormLabel color="gray.300">Email Address</FormLabel>
                            <Input
                                type="email"
                                value={email}
                                onChange={(e) => setEmail(e.target.value)}
                                bg="gray.700"
                                border="none"
                                color="white"
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
                            />
                        </FormControl>

                        <Button
                            type="submit"
                            colorScheme="green"
                            w="100%"
                            isLoading={isLoading}
                            mt={4}
                        >
                            Sign Up
                        </Button>

                        <Text fontSize="sm" color="gray.400">
                            Already have an account?{' '}
                            <Link as={RouterLink} to="/login" color="blue.400">
                                Log In
                            </Link>
                        </Text>
                    </VStack>
                </Box>
            </Container>
        </Box>
    );
};

export default Register;
