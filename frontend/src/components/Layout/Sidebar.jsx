import React from 'react';
import { Box, VStack, Link, Icon, Text, Divider } from '@chakra-ui/react';
import { FiHome, FiMusic, FiList, FiDisc, FiUploadCloud } from 'react-icons/fi';
import { Link as RouterLink, useLocation } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import { useDroppable } from '@dnd-kit/core';

const NavItem = ({ icon, children, to }) => {
    const location = useLocation();
    const isActive = location.pathname === to;

    return (
        <Link
            as={RouterLink}
            to={to}
            style={{ textDecoration: 'none' }}
            _focus={{ boxShadow: 'none' }}
        >
            <Box
                p={3}
                cursor="pointer"
                bg={isActive ? 'gray.700' : 'transparent'}
                _hover={{ bg: 'gray.800', color: 'white' }}
                display="flex"
                alignItems="center"
                borderLeft={isActive ? '4px solid' : '4px solid transparent'}
                borderColor={isActive ? 'blue.400' : 'transparent'}
            >
                <Icon as={icon} mr={3} boxSize={5} />
                <Text fontWeight="bold">{children}</Text>
            </Box>
        </Link>
    );
};

const DroppablePlaylistLink = ({ pl }) => {
    const { isOver, setNodeRef } = useDroppable({
        id: `playlist-${pl.id}`,
        data: { type: 'playlist', id: pl.id, name: pl.name }
    });

    return (
        <Link
            ref={setNodeRef}
            as={RouterLink}
            to={`/playlists/${pl.id}`}
            p={2}
            fontSize="sm"
            _hover={{ color: 'white', bg: 'gray.800' }}
            borderRadius="md"
            color={window.location.pathname === `/playlists/${pl.id}` ? 'white' : 'gray.500'}
            bg={isOver ? 'blue.900' : (window.location.pathname === `/playlists/${pl.id}` ? 'gray.800' : 'transparent')}
            border={isOver ? '1px dashed' : 'none'}
            borderColor="blue.400"
        >
            {pl.name}
        </Link>
    );
};

const Sidebar = () => {
    const { token } = useAuth();
    const [playlists, setPlaylists] = React.useState([]);

    React.useEffect(() => {
        const fetchPlaylists = async () => {
            if (!token) return;
            try {
                const res = await fetch('/api/playlists', {
                    headers: { 'Authorization': `Bearer ${token}` }
                });
                if (res.ok) {
                    const data = await res.json();
                    setPlaylists(data || []);
                }
            } catch (error) {
                console.error("Failed to fetch playlists for sidebar", error);
            }
        };
        fetchPlaylists();
    }, [token]);

    return (
        <Box
            w="250px"
            h="100%"
            bg="gray.900"
            color="gray.400"
            borderRight="1px solid"
            borderColor="gray.800"
            py={5}
            overflowY="auto"
        >
            <Box px={5} mb={8}>
                <Text fontSize="2xl" fontWeight="bold" color="white" letterSpacing="tight">
                    BIOMUZAK
                </Text>
            </Box>

            <VStack align="stretch" spacing={1}>
                <Text px={5} fontSize="xs" fontWeight="bold" textTransform="uppercase" letterSpacing="widest" mb={2}>
                    Menu
                </Text>
                <NavItem icon={FiHome} to="/">Home</NavItem>
                <NavItem icon={FiMusic} to="/library">Library</NavItem>

                <Divider my={4} borderColor="gray.700" />

                <Text px={5} fontSize="xs" fontWeight="bold" textTransform="uppercase" letterSpacing="widest" mb={2}>
                    Your Collection
                </Text>
                <NavItem icon={FiList} to="/playlists">Playlists</NavItem>

                {/* Exploded Playlist Submenu */}
                <VStack align="stretch" spacing={0} pl={4} mb={2}>
                    {playlists.map(pl => (
                        <DroppablePlaylistLink key={pl.id} pl={pl} />
                    ))}
                </VStack>

                <NavItem icon={FiUploadCloud} to="/upload">Upload</NavItem>
            </VStack>
        </Box>
    );
};

export default Sidebar;
