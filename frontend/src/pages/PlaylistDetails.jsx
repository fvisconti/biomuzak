import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
    Box, Heading, Text, VStack, Container, Table, Thead, Tbody, Tr, Th, Td,
    HStack, Button, Icon, Spinner, IconButton, useToast, useColorModeValue
} from '@chakra-ui/react';
import { FiMusic, FiUpload, FiTrash2, FiArrowLeft, FiPlay, FiChevronUp, FiChevronDown, FiDownload } from 'react-icons/fi';
import { useAuth } from '../context/AuthContext';
import { usePlayer } from '../context/PlayerContext';
import { useDrag } from '../context/DragContext';
import { usePlaylists } from '../context/PlaylistContext';
import {
    closestCenter,
} from '@dnd-kit/core';
import {
    arrayMove,
    SortableContext,
    verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';

import SortableSongRow from '../components/Music/SortableSongRow';

const PlaylistDetails = () => {
    const { playlistID } = useParams();
    const navigate = useNavigate();
    const [playlist, setPlaylist] = useState(null);
    const [loading, setLoading] = useState(true);
    const [sortBy, setSortBy] = useState('position');
    const [sortOrder, setSortOrder] = useState('asc');
    const { token } = useAuth();
    const toast = useToast();
    const { playPlaylist } = usePlayer();
    const { setOnDragEnd } = useDrag();
    const { refreshPlaylists } = usePlaylists();
    const emptyBorderColor = useColorModeValue('gray.300', 'gray.700');

    const handleSort = (column) => {
        if (sortBy === column) {
            setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
        } else {
            setSortBy(column);
            setSortOrder('asc');
        }
    };

    const sortedSongs = React.useMemo(() => {
        if (!playlist || !playlist.songs) return [];
        const songs = [...playlist.songs];
        if (sortBy === 'position') return songs;

        return songs.sort((a, b) => {
            let valA = a[sortBy] || '';
            let valB = b[sortBy] || '';
            if (typeof valA === 'string') valA = valA.toLowerCase();
            if (typeof valB === 'string') valB = valB.toLowerCase();

            if (valA < valB) return sortOrder === 'asc' ? -1 : 1;
            if (valA > valB) return sortOrder === 'asc' ? 1 : -1;
            return 0;
        });
    }, [playlist, sortBy, sortOrder]);

    const handleDragEnd = async (event) => {
        const { active, over } = event;
        if (!over) return;

        // Internal reordering
        if (active.id !== over.id) {
            const oldIndex = playlist.songs.findIndex((s) => s.id === active.id);
            const newIndex = playlist.songs.findIndex((s) => s.id === over.id);

            if (oldIndex !== -1 && newIndex !== -1) {
                const newSongs = arrayMove(playlist.songs, oldIndex, newIndex);
                setPlaylist({ ...playlist, songs: newSongs });

                try {
                    await fetch(`/api/playlists/${playlistID}/reorder`, {
                        method: 'PUT',
                        headers: {
                            'Authorization': `Bearer ${token}`,
                            'Content-Type': 'application/json'
                        },
                        body: JSON.stringify({
                            song_id: active.id,
                            new_position: newIndex + 1
                        })
                    });
                } catch (error) {
                    toast({ title: "Failed to save order", status: "error" });
                }
            }
        }
    };

    // Register the drag handler
    useEffect(() => {
        setOnDragEnd(handleDragEnd);
        return () => setOnDragEnd(null);
    }, [handleDragEnd, setOnDragEnd]);

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
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [playlistID, token]);

    const handleUploadToPlaylist = () => {
        // Navigate to upload page with state to pre-select this playlist
        navigate('/upload', { state: { playlistID: parseInt(playlistID), playlistName: playlist?.name } });
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

    const handleDeletePlaylist = async () => {
        if (!window.confirm("Are you sure you want to delete this playlist? The songs will remain in your library.")) return;

        try {
            const res = await fetch(`/api/playlists/${playlistID}`, {
                method: 'DELETE',
                headers: { 'Authorization': `Bearer ${token}` }
            });
            if (res.ok) {
                toast({ title: "Playlist deleted", status: "success" });
                refreshPlaylists();
                navigate('/playlists');
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
                        <Button
                            leftIcon={<FiDownload />}
                            colorScheme="blue"
                            variant="outline"
                            onClick={() => {
                                const url = `/api/playlists/${playlistID}/download?token=${token}`;
                                window.open(url, '_blank');
                            }}
                        >
                            Download All
                        </Button>
                        <Button leftIcon={<FiUpload />} colorScheme="brand" onClick={handleUploadToPlaylist}>
                            Add Music
                        </Button>
                        <Button leftIcon={<FiTrash2 />} colorScheme="red" variant="outline" onClick={handleDeletePlaylist}>
                            Delete
                        </Button>
                    </HStack>
                </HStack>

                {(!playlist.songs || playlist.songs.length === 0) ? (
                    <Box
                        p={10}
                        border="1px dashed"
                        borderColor={emptyBorderColor}
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
                                    <Th w="40px"></Th>
                                    <Th cursor="pointer" onClick={() => handleSort('title')}>
                                        Title {sortBy === 'title' && (sortOrder === 'asc' ? <FiChevronUp /> : <FiChevronDown />)}
                                    </Th>
                                    <Th cursor="pointer" onClick={() => handleSort('artist')}>
                                        Artist {sortBy === 'artist' && (sortOrder === 'asc' ? <FiChevronUp /> : <FiChevronDown />)}
                                    </Th>
                                    <Th cursor="pointer" onClick={() => handleSort('album')}>
                                        Album {sortBy === 'album' && (sortOrder === 'asc' ? <FiChevronUp /> : <FiChevronDown />)}
                                    </Th>
                                    <Th cursor="pointer" onClick={() => handleSort('duration')}>
                                        Duration {sortBy === 'duration' && (sortOrder === 'asc' ? <FiChevronUp /> : <FiChevronDown />)}
                                    </Th>
                                    <Th></Th>
                                </Tr>
                            </Thead>
                            <Tbody>
                                <SortableContext
                                    items={sortedSongs.map(s => s.id)}
                                    strategy={verticalListSortingStrategy}
                                >
                                    {sortedSongs.map((song, index) => (
                                        <SortableSongRow
                                            key={song.id}
                                            song={song}
                                            index={playlist.songs.indexOf(song)}
                                            token={token}
                                            handlePlaySong={handlePlaySong}
                                            handleDeleteSong={handleDeleteSong}
                                        />
                                    ))}
                                </SortableContext>
                            </Tbody>
                        </Table>
                    </Box>
                )}
            </VStack>
        </Container>
    );
};

export default PlaylistDetails;
