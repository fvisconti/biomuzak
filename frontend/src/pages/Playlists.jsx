import React, { useState, useEffect } from 'react';
import {
    Box, Heading, Text, VStack, Button, Container, HStack, Spacer,
    Modal, ModalOverlay, ModalContent, ModalHeader, ModalFooter, ModalBody, ModalCloseButton,
    Input, useDisclosure, useToast, SimpleGrid, Icon, IconButton, useColorModeValue
} from '@chakra-ui/react';
import { FiPlus, FiUpload, FiList, FiTrash2 } from 'react-icons/fi';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { usePlaylists } from '../context/PlaylistContext';

const Playlists = () => {
    const navigate = useNavigate();
    const { isOpen, onOpen, onClose } = useDisclosure();
    const [playlistName, setPlaylistName] = useState('');
    const { playlists, loading, refreshPlaylists } = usePlaylists();
    const { token } = useAuth();
    const toast = useToast();

    const cardBg = useColorModeValue('white', 'gray.800');
    const borderColor = useColorModeValue('gray.200', 'gray.700');
    const modalBg = useColorModeValue('white', 'gray.900');
    const emptyBorderColor = useColorModeValue('gray.300', 'gray.700');

    // Playlist fetching is now handled by PlaylistContext

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
                refreshPlaylists(); // Refresh list
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

    const handleDeletePlaylist = async (id) => {
        if (!window.confirm("Are you sure you want to delete this playlist? The songs will remain in your library.")) return;

        try {
            const res = await fetch(`/api/playlists/${id}`, {
                method: 'DELETE',
                headers: {
                    'Authorization': `Bearer ${token}`
                }
            });

            if (res.ok) {
                toast({
                    title: "Playlist Deleted",
                    status: "success",
                    duration: 3000,
                    isClosable: true,
                });
                refreshPlaylists();
            } else {
                toast({
                    title: "Failed to delete playlist",
                    status: "error",
                    duration: 3000,
                    isClosable: true,
                });
            }
        } catch (error) {
            console.error("Error deleting playlist", error);
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
                        borderColor={emptyBorderColor}
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
                                bg={cardBg}
                                borderRadius="lg"
                                borderWidth="1px"
                                borderColor={borderColor}
                                _hover={{ borderColor: 'brand.500' }}
                                cursor="pointer"
                                onClick={() => navigate(`/playlists/${playlist.id}`)}
                            >
                                <HStack spacing={4} justify="space-between" width="100%">
                                    <HStack spacing={4} onClick={() => navigate(`/playlists/${playlist.id}`)}>
                                        <Icon as={FiList} boxSize={6} color="brand.400" />
                                        <VStack align="start" spacing={0}>
                                            <Text fontWeight="bold" fontSize="lg">{playlist.name}</Text>
                                            <Text fontSize="sm" color="gray.500">{playlist.song_count || 0} songs</Text>
                                        </VStack>
                                    </HStack>
                                    <IconButton
                                        icon={<FiTrash2 />}
                                        size="sm"
                                        colorScheme="red"
                                        variant="ghost"
                                        aria-label="Delete Playlist"
                                        onClick={(e) => {
                                            e.stopPropagation();
                                            handleDeletePlaylist(playlist.id);
                                        }}
                                    />
                                </HStack>
                            </Box>
                        ))}
                    </SimpleGrid>
                )}
            </VStack>

            <Modal isOpen={isOpen} onClose={onClose}>
                <ModalOverlay />
                <ModalContent bg={modalBg} borderColor={borderColor} borderWidth="1px">
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
