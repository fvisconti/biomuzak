import React, { useState, useEffect } from 'react';
import {
    Box, Heading, Text, Container, SimpleGrid, HStack, Button, Spacer,
    Spinner, useToast, VStack, Icon, useColorModeValue
} from '@chakra-ui/react';
import { FiUpload, FiDisc } from 'react-icons/fi';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

const AlbumCard = ({ album, onClick }) => {
    const bg = useColorModeValue('white', 'gray.800');
    const border = useColorModeValue('gray.200', 'gray.700');

    return (
        <Box
            p={4}
            bg={bg}
            border="1px"
            borderColor={border}
            borderRadius="lg"
            cursor="pointer"
            transition="all 0.2s"
            _hover={{ transform: 'translateY(-2px)', shadow: 'md' }}
            onClick={onClick}
        >
            <VStack spacing={3}>
                <Box
                    w="100%"
                    pt="100%"
                    position="relative"
                    bg="gray.100"
                    borderRadius="md"
                    overflow="hidden"
                >
                    <Icon
                        as={FiDisc}
                        position="absolute"
                        top="50%"
                        left="50%"
                        transform="translate(-50%, -50%)"
                        boxSize="40px"
                        color="gray.400"
                    />
                </Box>
                <VStack spacing={0} align="center" w="100%">
                    <Text fontWeight="bold" noOfLines={1} title={album.name}>
                        {album.name}
                    </Text>
                    <Text fontSize="sm" color="gray.500" noOfLines={1} title={album.artist}>
                        {album.artist || 'Unknown Artist'}
                    </Text>
                    <Text fontSize="xs" color="gray.400">
                        {album.song_count} songs • {album.year || '-'}
                    </Text>
                </VStack>
            </VStack>
        </Box>
    );
};

const Albums = () => {
    const navigate = useNavigate();
    const { token } = useAuth();
    const [albums, setAlbums] = useState([]);
    const [loading, setLoading] = useState(true);
    const toast = useToast();

    useEffect(() => {
        const fetchAlbums = async () => {
            try {
                const res = await fetch('/api/albums', {
                    headers: { 'Authorization': `Bearer ${token}` }
                });
                if (res.ok) {
                    const data = await res.json();
                    setAlbums(data || []);
                } else {
                    toast({ title: "Failed to load albums", status: "error" });
                }
            } catch (error) {
                console.error(error);
                toast({ title: "Error loading albums", status: "error" });
            } finally {
                setLoading(false);
            }
        };

        if (token) fetchAlbums();
    }, [token, toast]);

    const handleAlbumClick = (album) => {
        // Navigate to details page with query params
        navigate(`/albums/view?album=${encodeURIComponent(album.name)}&artist=${encodeURIComponent(album.artist)}`);
    };

    if (loading) {
        return (
            <Container maxW="container.xl" py={8} centerContent>
                <Spinner size="xl" />
            </Container>
        );
    }

    return (
        <Container maxW="container.xl" py={8} pb={24}>
            <HStack mb={6}>
                <Heading as="h1" size="xl">
                    Albums
                </Heading>
                <Spacer />
                <Button
                    leftIcon={<FiUpload />}
                    colorScheme="brand"
                    variant="solid"
                    onClick={() => navigate('/upload')}
                >
                    Upload to Library
                </Button>
            </HStack>

            {albums.length === 0 ? (
                <Box
                    p={10}
                    border="1px dashed"
                    borderColor="gray.700"
                    borderRadius="md"
                    textAlign="center"
                >
                    <Text color="gray.500" fontSize="lg">
                        No albums found. Upload some music to get started.
                    </Text>
                </Box>
            ) : (
                <SimpleGrid columns={{ base: 2, md: 3, lg: 4, xl: 5 }} spacing={6}>
                    {albums.map((album, idx) => (
                        <AlbumCard
                            key={`${album.name}-${album.artist}-${idx}`}
                            album={album}
                            onClick={() => handleAlbumClick(album)}
                        />
                    ))}
                </SimpleGrid>
            )}
        </Container>
    );
};

export default Albums;
