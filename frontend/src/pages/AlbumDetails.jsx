import React, { useState, useEffect } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import {
    Box, Heading, Text, VStack, Container, Table, Thead, Tbody, Tr, Th, Td,
    HStack, Button, Icon, Spinner, IconButton, useToast, useColorModeValue
} from '@chakra-ui/react';
import { FiMusic, FiArrowLeft, FiPlay, FiClock } from 'react-icons/fi';
import { useAuth } from '../context/AuthContext';
import { usePlayer } from '../context/PlayerContext';

const AlbumDetails = () => {
    const [searchParams] = useSearchParams();
    const navigate = useNavigate();
    const albumName = searchParams.get('album');
    const artistName = searchParams.get('artist');

    const [songs, setSongs] = useState([]);
    const [loading, setLoading] = useState(true);
    const { token } = useAuth();
    const { playPlaylist } = usePlayer();
    const toast = useToast();

    const bg = useColorModeValue('white', 'gray.800');

    useEffect(() => {
        const fetchAlbumSongs = async () => {
            if (!albumName) return;

            try {
                const query = `album=${encodeURIComponent(albumName)}&artist=${encodeURIComponent(artistName || '')}`;
                const res = await fetch(`/api/albums/songs?${query}`, {
                    headers: { 'Authorization': `Bearer ${token}` }
                });

                if (res.ok) {
                    const data = await res.json();
                    setSongs(data || []);
                } else {
                    toast({ title: "Failed to load album songs", status: "error" });
                }
            } catch (error) {
                console.error(error);
                toast({ title: "Error loading songs", status: "error" });
            } finally {
                setLoading(false);
            }
        };

        if (token) fetchAlbumSongs();
    }, [albumName, artistName, token, toast]);

    const handlePlayAll = () => {
        if (songs.length > 0) {
            playPlaylist(songs);
        }
    };

    const handlePlaySong = (index) => {
        playPlaylist(songs, index);
    };

    if (loading) return <Spinner size="xl" mt={10} ml={10} />;

    return (
        <Container maxW="container.xl" py={8} pb={24}>
            <VStack spacing={6} align="stretch">
                <Button
                    leftIcon={<FiArrowLeft />}
                    variant="ghost"
                    alignSelf="start"
                    onClick={() => navigate('/albums')}
                >
                    Back to Albums
                </Button>

                <HStack align="start" spacing={6} p={6} bg={bg} borderRadius="lg" shadow="sm">
                    {/* Placeholder Cover Art */}
                    <Box
                        w="150px"
                        h="150px"
                        bg="gray.300"
                        borderRadius="md"
                        display="flex"
                        alignItems="center"
                        justifyContent="center"
                    >
                        <Icon as={FiMusic} boxSize="60px" color="gray.500" />
                    </Box>

                    <VStack align="start" spacing={2} flex={1}>
                        <Heading as="h1" size="xl">{albumName}</Heading>
                        <Text fontSize="xl" color="gray.500">{artistName || 'Unknown Artist'}</Text>
                        <Text fontSize="sm" color="gray.400">
                            {songs.length} songs • {songs.length > 0 ? (songs[0].year || '-') : '-'}
                        </Text>
                        <Button
                            leftIcon={<FiPlay />}
                            colorScheme="brand"
                            size="lg"
                            mt={2}
                            onClick={handlePlayAll}
                            isDisabled={songs.length === 0}
                        >
                            Play Album
                        </Button>
                    </VStack>
                </HStack>

                <Box overflowX="auto" bg={bg} borderRadius="lg" shadow="sm">
                    <Table variant="simple">
                        <Thead>
                            <Tr>
                                <Th w="50px">#</Th>
                                <Th>Title</Th>
                                <Th>Artist</Th>
                                <Th isNumeric><Icon as={FiClock} /></Th>
                            </Tr>
                        </Thead>
                        <Tbody>
                            {songs.map((song, index) => (
                                <Tr key={song.id} _hover={{ bg: 'whiteAlpha.50' }}>
                                    <Td>
                                        <HStack>
                                            <Text color="gray.500" fontSize="sm" w="20px">{index + 1}</Text>
                                            <IconButton
                                                icon={<FiPlay />}
                                                size="xs"
                                                variant="ghost"
                                                colorScheme="brand"
                                                onClick={() => handlePlaySong(index)}
                                                aria-label="Play song"
                                            />
                                        </HStack>
                                    </Td>
                                    <Td fontWeight="medium">{song.title}</Td>
                                    <Td>{song.artist}</Td>
                                    <Td isNumeric fontFamily="monospace">
                                        {Math.floor(song.duration / 60)}:{(song.duration % 60).toString().padStart(2, '0')}
                                    </Td>
                                </Tr>
                            ))}
                        </Tbody>
                    </Table>
                </Box>
            </VStack>
        </Container>
    );
};

export default AlbumDetails;
