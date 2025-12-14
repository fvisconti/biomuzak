import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
    Box, Heading, Text, VStack, Container, Table, Thead, Tbody, Tr, Th, Td,
    HStack, Button, Icon, Spinner, IconButton, useToast
} from '@chakra-ui/react';
import { FiMusic, FiUpload, FiTrash2, FiArrowLeft, FiPlay } from 'react-icons/fi';
import { useAuth } from '../context/AuthContext';
import { usePlayer } from '../context/PlayerContext';

const PlaylistDetails = () => {
    const { playlistID } = useParams();
    const navigate = useNavigate();
    const [playlist, setPlaylist] = useState(null);
    const [loading, setLoading] = useState(true);
    const { token } = useAuth();
    const toast = useToast();
    const { playPlaylist } = usePlayer();

    const fetchPlaylist = async () => {
        try {
            const res = await fetch(`/api/playlists/${playlistID}`, {
                headers: { 'Authorization': `Bearer ${token}` }
            });
            if (res.ok) {
                const data = await res.json();
                setPlaylist(data);
            } else {
                toast({ title: "Failed to load playlist", status: "error" });
                navigate('/playlists');
            }
        } catch (error) {
            console.error(error);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchPlaylist();
    }, [playlistID, token]);

    const handleUploadToPlaylist = () => {
        // Navigate to upload page with state to pre-select this playlist
        navigate('/upload', { state: { playlistID: parseInt(playlistID), playlistName: playlist.name } });
    };

    const handleDeleteSong = async (songID) => {
        try {
            const res = await fetch(`/api/playlists/${playlistID}/songs/${songID}`, {
                method: 'DELETE',
                headers: { 'Authorization': `Bearer ${token}` }
            });
            if (res.ok) {
                toast({ title: "Song removed", status: "success" });
                fetchPlaylist();
            }
        } catch (error) {
            console.error(error);
        }
    };

    const handlePlayAll = () => {
        if (playlist && playlist.songs && playlist.songs.length > 0) {
            playPlaylist(playlist.songs);
        }
    };

    const handlePlaySong = (index) => {
        if (playlist && playlist.songs) {
            playPlaylist(playlist.songs, index);
        }
    };

    if (loading) return <Spinner size="xl" mt={10} ml={10} />;
    if (!playlist) return null;

    return (
        <Container maxW="container.xl" py={8} pb={24}>
            <VStack spacing={6} align="stretch">
                <Button
                    leftIcon={<FiArrowLeft />}
                    variant="ghost"
                    alignSelf="start"
                    onClick={() => navigate('/playlists')}
                >
                    Back to Playlists
                </Button>

                <HStack justify="space-between">
                    <VStack align="start" spacing={1}>
                        <Heading as="h1" size="xl">{playlist.name}</Heading>
                        <Text color="gray.500">{playlist.songs ? playlist.songs.length : 0} songs</Text>
                    </VStack>
                    <HStack>
                        <Button leftIcon={<FiPlay />} colorScheme="green" onClick={handlePlayAll}>
                            Play All
                        </Button>
                        <Button leftIcon={<FiUpload />} colorScheme="brand" onClick={handleUploadToPlaylist}>
                            Add Music
                        </Button>
                    </HStack>
                </HStack>

                {(!playlist.songs || playlist.songs.length === 0) ? (
                    <Box
                        p={10}
                        border="1px dashed"
                        borderColor="gray.700"
                        borderRadius="md"
                        textAlign="center"
                    >
                        <Text color="gray.500" fontSize="lg">
                            This playlist is empty.
                        </Text>
                        <Button mt={4} colorScheme="brand" variant="link" onClick={handleUploadToPlaylist}>
                            Upload songs to this playlist
                        </Button>
                    </Box>
                ) : (
                    <Box overflowX="auto">
                        <Table variant="simple" size="sm">
                            <Thead>
                                <Tr>
                                    <Th>#</Th>
                                    <Th>Title</Th>
                                    <Th>Artist</Th>
                                    <Th>Album</Th>
                                    <Th>Duration</Th>
                                    <Th></Th>
                                </Tr>
                            </Thead>
                            <Tbody>
                                {playlist.songs.map((song, index) => (
                                    <Tr key={song.id} _hover={{ bg: 'whiteAlpha.50' }}>
                                        <Td>
                                            <IconButton
                                                icon={<FiPlay />}
                                                size="xs"
                                                variant="ghost"
                                                colorScheme="brand"
                                                onClick={() => handlePlaySong(index)}
                                                aria-label="Play"
                                            />
                                        </Td>
                                        <Td fontWeight="bold">
                                            <HStack>
                                                <Icon as={FiMusic} color="brand.400" />
                                                <Text>{song.title || 'Unknown Title'}</Text>
                                            </HStack>
                                        </Td>
                                        <Td>{song.artist || '-'}</Td>
                                        <Td>{song.album || '-'}</Td>
                                        <Td fontFamily="monospace">
                                            {Math.floor(song.duration / 60)}:{(song.duration % 60).toString().padStart(2, '0')}
                                        </Td>
                                        <Td>
                                            <IconButton
                                                icon={<FiTrash2 />}
                                                size="sm"
                                                colorScheme="red"
                                                variant="ghost"
                                                onClick={() => handleDeleteSong(song.id)}
                                                aria-label="Remove song"
                                            />
                                        </Td>
                                    </Tr>
                                ))}
                            </Tbody>
                        </Table>
                    </Box>
                )}
            </VStack>
        </Container>
    );
};

export default PlaylistDetails;
