import React, { useState, useEffect } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import {
    Box, Heading, VStack, Container, Table, Thead, Tbody, Tr, Th, Td,
    Badge, HStack, Icon, IconButton, Spinner, Menu, MenuButton, MenuList, MenuItem,
    Modal, ModalOverlay, ModalContent, ModalHeader, ModalBody, ModalFooter,
    Button, Select, Input, useToast, Editable, EditablePreview, EditableInput, Text, useColorModeValue
} from '@chakra-ui/react';
import { FiMusic, FiRefreshCw, FiPlay, FiMoreVertical, FiTrash2, FiPlusCircle, FiDownload } from 'react-icons/fi';
import { useAuth } from '../context/AuthContext';
import { usePlayer } from '../context/PlayerContext';
import { usePlaylists } from '../context/PlaylistContext';
import { useDraggable } from '@dnd-kit/core';
import { CSS } from '@dnd-kit/utilities';

const DraggableSongRow = ({ song, songs, token, playPlaylist, handleGenreUpdate, setSelectedSong, setShowAddToPlaylist, handleDeleteSong, formatDuration }) => {
    const { attributes, listeners, setNodeRef, transform } = useDraggable({
        id: `track-${song.id}`,
        data: { id: song.id, ...song },
    });

    const style = {
        transform: CSS.Translate.toString(transform),
    };

    return (
        <Tr
            key={song.id}
            ref={setNodeRef}
            style={style}
            {...listeners}
            {...attributes}
            _hover={{ bg: 'whiteAlpha.50', cursor: 'grab' }}
            bg={transform ? 'gray.700' : 'transparent'}
        >
            <Td>
                <IconButton
                    icon={<FiPlay />}
                    size="xs"
                    colorScheme="brand"
                    variant="ghost"
                    aria-label="Play song"
                    onClick={(e) => { e.stopPropagation(); playPlaylist(songs, songs.indexOf(song)); }}
                    onPointerDown={(e) => e.stopPropagation()}
                />
            </Td>
            <Td fontWeight="bold">
                <Text>{song.title || 'Unknown Title'}</Text>
            </Td>
            <Td>{song.artist || 'Unknown Artist'}</Td>
            <Td>{song.album || 'Unknown Album'}</Td>
            <Td>{song.year || '-'}</Td>
            <Td>
                <Editable
                    defaultValue={song.genre || 'Unknown'}
                    onSubmit={(newGenre) => handleGenreUpdate(song.id, newGenre)}
                >
                    <EditablePreview
                        as={Badge}
                        colorScheme="purple"
                        variant="subtle"
                        cursor="pointer"
                        _hover={{ bg: 'purple.600' }}
                    />
                    <EditableInput />
                </Editable>
            </Td>
            <Td isNumeric fontFamily="monospace">{formatDuration(song.duration)}</Td>
            <Td>
                <Menu>
                    <MenuButton
                        as={IconButton}
                        icon={<FiMoreVertical />}
                        variant="ghost"
                        size="sm"
                        aria-label="Actions"
                        onPointerDown={(e) => e.stopPropagation()}
                    />
                    <MenuList>
                        <MenuItem
                            icon={<FiPlusCircle />}
                            onClick={() => {
                                setSelectedSong(song);
                                setShowAddToPlaylist(true);
                            }}
                        >
                            Add to Playlist
                        </MenuItem>
                        <MenuItem
                            icon={<FiDownload />}
                            onClick={() => {
                                const url = `/api/songs/${song.id}/download?token=${token}`;
                                window.open(url, '_blank');
                            }}
                        >
                            Download Song
                        </MenuItem>
                        <MenuItem
                            icon={<FiTrash2 />}
                            onClick={() => handleDeleteSong(song.id)}
                            color="red.400"
                        >
                            Delete Song
                        </MenuItem>
                    </MenuList>
                </Menu>
            </Td>
        </Tr>
    );
};

const Library = () => {
    const navigate = useNavigate();
    const location = useLocation();
    const queryParams = new URLSearchParams(location.search);
    const searchQuery = queryParams.get('q') || '';

    const [songs, setSongs] = useState([]);
    const { playlists, refreshPlaylists } = usePlaylists();
    const [loading, setLoading] = useState(true);
    const [sortBy, setSortBy] = useState('title');
    const [showAddToPlaylist, setShowAddToPlaylist] = useState(false);
    const [selectedSong, setSelectedSong] = useState(null);
    const [newPlaylistName, setNewPlaylistName] = useState('');
    const { token } = useAuth();
    const { playPlaylist } = usePlayer();
    const toast = useToast();
    const modalBg = useColorModeValue('white', 'gray.800');
    const emptyBorderColor = useColorModeValue('gray.300', 'gray.700');

    const fetchLibrary = async () => {
        setLoading(true);
        try {
            const url = new URL('/api/library', window.location.origin);
            url.searchParams.append('sort_by', sortBy);
            if (searchQuery) {
                url.searchParams.append('q', searchQuery);
            }

            const res = await fetch(url, {
                headers: { 'Authorization': `Bearer ${token}` }
            });
            if (res.ok) {
                const data = await res.json();
                setSongs(data || []);
            }
        } catch (error) {
            console.error("Failed to fetch library", error);
        } finally {
            setLoading(false);
        }
    };

    // Playlist fetching is now handled by PlaylistContext

    useEffect(() => {
        fetchLibrary();
    }, [token, sortBy, searchQuery]);

    const formatDuration = (seconds) => {
        const mins = Math.floor(seconds / 60);
        const secs = seconds % 60;
        return `${mins}:${secs.toString().padStart(2, '0')}`;
    };

    const handleDeleteSong = async (songId) => {
        if (!confirm('Remove this song from your library?')) return;

        try {
            const res = await fetch(`/api/songs/${songId}`, {
                method: 'DELETE',
                headers: { 'Authorization': `Bearer ${token}` }
            });
            if (res.ok) {
                toast({ title: "Song removed", status: "success", duration: 2000 });
                fetchLibrary();
            }
        } catch (error) {
            toast({ title: "Failed to delete song", status: "error", duration: 2000 });
        }
    };

    const handleAddToPlaylist = async (playlistId) => {
        try {
            const res = await fetch(`/api/playlists/${playlistId}/songs`, {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ song_id: selectedSong.id })
            });
            if (res.ok) {
                toast({ title: "Added to playlist", status: "success", duration: 2000 });
                setShowAddToPlaylist(false);
            }
        } catch (error) {
            toast({ title: "Failed to add to playlist", status: "error", duration: 2000 });
        }
    };

    const handleCreateAndAddToPlaylist = async () => {
        if (!newPlaylistName.trim()) return;

        try {
            // Create playlist
            const createRes = await fetch('/api/playlists', {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ name: newPlaylistName })
            });

            if (createRes.ok) {
                const playlist = await createRes.json();
                await handleAddToPlaylist(playlist.id);
                setNewPlaylistName('');
                refreshPlaylists();
            }
        } catch (error) {
            toast({ title: "Failed to create playlist", status: "error", duration: 2000 });
        }
    };

    const handleGenreUpdate = async (songId, newGenre) => {
        try {
            const res = await fetch(`/api/songs/${songId}/genre`, {
                method: 'PUT',
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ genre: newGenre })
            });
            if (res.ok) {
                toast({ title: "Genre updated", status: "success", duration: 2000 });
                fetchLibrary();
            }
        } catch (error) {
            toast({ title: "Failed to update genre", status: "error", duration: 2000 });
        }
    };

    return (
        <Container maxW="container.xl" py={8}>
            <VStack spacing={6} align="stretch">
                <HStack justify="space-between">
                    <Heading as="h1" size="xl">
                        {searchQuery ? `Search Results for "${searchQuery}"` : `Library (${songs.length})`}
                    </Heading>
                    <IconButton
                        icon={<FiRefreshCw />}
                        onClick={fetchLibrary}
                        aria-label="Refresh Library"
                        variant="ghost"
                    />
                </HStack>

                {loading ? (
                    <Spinner size="xl" alignSelf="center" mt={10} />
                ) : songs.length === 0 ? (
                    <Box
                        p={10}
                        border="1px dashed"
                        borderColor={emptyBorderColor}
                        borderRadius="md"
                        textAlign="center"
                    >
                        <Text color="gray.500" fontSize="lg">
                            Your library is empty. Upload some music!
                        </Text>
                    </Box>
                ) : (
                    <Box overflowX="auto">
                        <Table variant="simple" size="sm">
                            <Thead>
                                <Tr>
                                    <Th w="40px"></Th>
                                    <Th cursor="pointer" onClick={() => setSortBy('title')}>
                                        Title {sortBy === 'title' && '↓'}
                                    </Th>
                                    <Th cursor="pointer" onClick={() => setSortBy('artist')}>
                                        Artist {sortBy === 'artist' && '↓'}
                                    </Th>
                                    <Th cursor="pointer" onClick={() => setSortBy('album')}>
                                        Album {sortBy === 'album' && '↓'}
                                    </Th>
                                    <Th cursor="pointer" onClick={() => setSortBy('year')}>
                                        Year {sortBy === 'year' && '↓'}
                                    </Th>
                                    <Th>Genre</Th>
                                    <Th isNumeric cursor="pointer" onClick={() => setSortBy('duration')}>
                                        Duration {sortBy === 'duration' && '↓'}
                                    </Th>
                                    <Th>Action</Th>
                                </Tr>
                            </Thead>
                            <Tbody>
                                {songs.map((song) => (
                                    <DraggableSongRow
                                        key={song.id}
                                        song={song}
                                        songs={songs}
                                        token={token}
                                        playPlaylist={playPlaylist}
                                        handleGenreUpdate={handleGenreUpdate}
                                        setSelectedSong={setSelectedSong}
                                        setShowAddToPlaylist={setShowAddToPlaylist}
                                        handleDeleteSong={handleDeleteSong}
                                        formatDuration={formatDuration}
                                    />
                                ))}
                            </Tbody>
                        </Table>
                    </Box>
                )}
            </VStack>

            <Modal isOpen={showAddToPlaylist} onClose={() => setShowAddToPlaylist(false)}>
                <ModalOverlay />
                <ModalContent bg={modalBg}>
                    <ModalHeader>Add to Playlist</ModalHeader>
                    <ModalBody>
                        <VStack spacing={4} align="stretch">
                            <Box>
                                <Text mb={2} fontSize="sm" color="gray.400">Select existing playlist:</Text>
                                {playlists.map((playlist) => (
                                    <Button
                                        key={playlist.id}
                                        w="full"
                                        mb={2}
                                        variant="outline"
                                        onClick={() => handleAddToPlaylist(playlist.id)}
                                    >
                                        {playlist.name}
                                    </Button>
                                ))}
                            </Box>
                            <Box>
                                <Text mb={2} fontSize="sm" color="gray.400">Or create new:</Text>
                                <HStack>
                                    <Input
                                        placeholder="New playlist name"
                                        value={newPlaylistName}
                                        onChange={(e) => setNewPlaylistName(e.target.value)}
                                    />
                                    <Button colorScheme="brand" onClick={handleCreateAndAddToPlaylist}>
                                        Create
                                    </Button>
                                </HStack>
                            </Box>
                        </VStack>
                    </ModalBody>
                    <ModalFooter>
                        <Button variant="ghost" onClick={() => setShowAddToPlaylist(false)}>
                            Cancel
                        </Button>
                    </ModalFooter>
                </ModalContent>
            </Modal>
        </Container>
    );
};

export default Library;
