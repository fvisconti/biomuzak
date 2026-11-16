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
    arrayMove,
    SortableContext,
    verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';

import SortableSongRow from '../components/Music/SortableSongRow';
import apiClient from '../api/client';

const PlaylistDetails = () => {
    const { playlistID } = useParams();
    const navigate = useNavigate();
    const [playlist, setPlaylist] = useState(null);
    const [loading, setLoading] = useState(true);
    // Sort is server-side and persisted per playlist (E6): the playlist's
    // stored sort_key decides the order; clicking a column header persists a
    // new key. added_at (most recent first) is the default.
    const [sortKey, setSortKey] = useState('added_at');
    const { token } = useAuth();
    const toast = useToast();
    const { playPlaylist } = usePlayer();
    const { setOnDragEnd } = useDrag();
    const { refreshPlaylists } = usePlaylists();
    const emptyBorderColor = useColorModeValue('gray.300', 'gray.700');

    const handleSort = async (column) => {
        if (sortKey === column) return; // direction is fixed server-side
        setSortKey(column);
        setPlaylist((p) => (p ? { ...p, sort_key: column } : p));
        try {
            await apiClient.put(`/playlists/${playlistID}`, { sort_key: column });
            fetchPlaylist();
        } catch {
            toast({ title: "Failed to save sort order", status: "error" });
        }
    };

    // The server returns songs in the persisted order already.
    const sortedSongs = playlist && playlist.songs ? playlist.songs : [];

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
                    // Persist the whole displayed order (the playlist may have
                    // been sorted, so a single-song move would not match what
                    // the user just arranged), then switch to position order.
                    await apiClient.put(`/playlists/${playlistID}/order`, {
                        song_ids: newSongs.map((s) => s.id)
                    });
                    await apiClient.put(`/playlists/${playlistID}`, { sort_key: 'position' });
                    setSortKey('position');
                } catch {
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
            // No sort override: the server applies the playlist's persisted
            // sort key.
            const res = await apiClient.get(`/playlists/${playlistID}`);
            const data = res.data;
            setPlaylist(data);
            if (data.sort_key) setSortKey(data.sort_key);
        } catch (error) {
            console.error(error);
            if (error.response && error.response.status === 404) {
                navigate('/playlists');
            } else {
                toast({ title: "Failed to load playlist", status: "error" });
            }
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
                                        Title {sortKey === 'title' && <FiChevronUp />}
                                    </Th>
                                    <Th cursor="pointer" onClick={() => handleSort('artist')}>
                                        Artist {sortKey === 'artist' && <FiChevronUp />}
                                    </Th>
                                    <Th>Album</Th>
                                    <Th>Duration</Th>
                                    <Th cursor="pointer" onClick={() => handleSort('added_at')}>
                                        Added {sortKey === 'added_at' && <FiChevronDown />}
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
                                            index={index}
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
