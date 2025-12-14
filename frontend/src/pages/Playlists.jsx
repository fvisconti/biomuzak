import React, { useState, useEffect } from 'react';
import {
    Box, Heading, Text, VStack, Button, Container, HStack, Spacer,
    Modal, ModalOverlay, ModalContent, ModalHeader, ModalFooter, ModalBody, ModalCloseButton,
    Input, useDisclosure, useToast, SimpleGrid, Icon
} from '@chakra-ui/react';
import { FiPlus, FiUpload, FiList } from 'react-icons/fi';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

const Playlists = () => {
    const navigate = useNavigate();
    const { isOpen, onOpen, onClose } = useDisclosure();
    const [playlistName, setPlaylistName] = useState('');
    const [playlists, setPlaylists] = useState([]);
    const [loading, setLoading] = useState(true);
    const { token } = useAuth();
    const toast = useToast();

    const fetchPlaylists = async () => {
        try {
            const res = await fetch('/api/playlists', {
                headers: {
                    'Authorization': `Bearer ${token}`
                }
            });
            if (res.ok) {
                const data = await res.json();
                setPlaylists(data || []);
            }
        } catch (error) {
            console.error("Failed to fetch playlists", error);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchPlaylists();
    }, [token]);

    const handleCreatePlaylist = async () => {
        try {
            const res = await fetch('/api/playlists', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${token}`
                },
                body: JSON.stringify({ name: playlistName })
            });

            if (res.ok) {
                toast({
                    title: "Playlist Created",
                    status: "success",
                    duration: 3000,
                    isClosable: true,
                });
                setPlaylistName('');
                onClose();
                fetchPlaylists(); // Refresh list
            } else {
                toast({
                    title: "Failed to create playlist",
                    status: "error",
                    duration: 3000,
                    isClosable: true,
                });
            }
        } catch (error) {
            console.error("Error creating playlist", error);
        }
    };

    return (
        <Container maxW="container.xl" py={8}>
            <VStack spacing={6} align="stretch">
                <HStack>
                    <Heading as="h1" size="xl">
                        Playlists
                    </Heading>
                    <Spacer />
                    <Button leftIcon={<FiPlus />} colorScheme="brand" variant="outline" onClick={onOpen}>
                        New Playlist
                    </Button>
                    <Button
                        leftIcon={<FiUpload />}
                        colorScheme="brand"
                        variant="solid"
                        onClick={() => navigate('/upload')}
                    >
                        Upload to Library
                    </Button>
                </HStack>

                {playlists.length === 0 ? (
                    <Box
                        p={10}
                        border="1px dashed"
                        borderColor="gray.700"
                        borderRadius="md"
                        textAlign="center"
                    >
                        <Text color="gray.500" fontSize="lg">
                            You haven't created any playlists yet.
                        </Text>
                        <Button mt={4} colorScheme="brand" variant="link" onClick={onOpen}>
                            Create your first playlist
                        </Button>
                    </Box>
                ) : (
                    <SimpleGrid columns={{ base: 1, md: 2, lg: 3 }} spacing={6}>
                        {playlists.map((playlist) => (
                            <Box
                                key={playlist.id}
                                p={5}
                                bg="gray.800"
                                borderRadius="lg"
                                borderWidth="1px"
                                borderColor="gray.700"
                                _hover={{ borderColor: 'brand.500' }}
                                cursor="pointer"
                                onClick={() => navigate(`/playlists/${playlist.id}`)}
                            >
                                <HStack spacing={4}>
                                    <Icon as={FiList} boxSize={6} color="brand.400" />
                                    <VStack align="start" spacing={0}>
                                        <Text fontWeight="bold" fontSize="lg">{playlist.name}</Text>
                                        <Text fontSize="sm" color="gray.500">{playlist.song_count || 0} songs</Text>
                                    </VStack>
                                </HStack>
                            </Box>
                        ))}
                    </SimpleGrid>
                )}
            </VStack>

            <Modal isOpen={isOpen} onClose={onClose}>
                <ModalOverlay />
                <ModalContent bg="gray.900" borderColor="gray.700" borderWidth="1px">
                    <ModalHeader>Create New Playlist</ModalHeader>
                    <ModalCloseButton />
                    <ModalBody>
                        <Input
                            placeholder="Playlist Name"
                            value={playlistName}
                            onChange={(e) => setPlaylistName(e.target.value)}
                            focusBorderColor="brand.500"
                        />
                    </ModalBody>

                    <ModalFooter>
                        <Button variant="ghost" mr={3} onClick={onClose}>
                            Cancel
                        </Button>
                        <Button colorScheme="brand" onClick={handleCreatePlaylist} isDisabled={!playlistName.trim()}>
                            Create
                        </Button>
                    </ModalFooter>
                </ModalContent>
            </Modal>
        </Container>
    );
};

export default Playlists;
