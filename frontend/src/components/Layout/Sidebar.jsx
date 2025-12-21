import React from 'react';
import { Box, VStack, Link, Icon, Text, Divider, useColorModeValue } from '@chakra-ui/react';
import { FiHome, FiMusic, FiList, FiDisc, FiUploadCloud } from 'react-icons/fi';
import { Link as RouterLink, useLocation } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import { usePlaylists } from '../../context/PlaylistContext';
import { useDroppable } from '@dnd-kit/core';

const NavItem = ({ icon, children, to }) => {
    const location = useLocation();
    const isActive = location.pathname === to;
    const hoverBg = useColorModeValue('gray.100', 'gray.800');
    const activeBg = useColorModeValue('gray.200', 'gray.700');
    const activeTextColor = useColorModeValue('blue.600', 'white');

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
                bg={isActive ? activeBg : 'transparent'}
                _hover={{ bg: hoverBg, color: activeTextColor }}
                display="flex"
                alignItems="center"
                borderLeft={isActive ? '4px solid' : '4px solid transparent'}
                borderColor={isActive ? 'blue.400' : 'transparent'}
                color={isActive ? activeTextColor : 'inherit'}
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

    const isActive = window.location.pathname === `/playlists/${pl.id}`;
    const activeColor = useColorModeValue('blue.600', 'white');
    const inactiveColor = useColorModeValue('gray.600', 'gray.500');
    const hoverBg = useColorModeValue('gray.100', 'gray.800');
    const activeBg = useColorModeValue('gray.200', 'gray.800');

    return (
        <Link
            ref={setNodeRef}
            as={RouterLink}
            to={`/playlists/${pl.id}`}
            p={2}
            fontSize="sm"
            _hover={{ color: activeColor, bg: hoverBg }}
            borderRadius="md"
            color={isActive ? activeColor : inactiveColor}
            bg={isOver ? 'blue.900' : (isActive ? activeBg : 'transparent')}
            border={isOver ? '1px dashed' : 'none'}
            borderColor="blue.400"
        >
            {pl.name}
        </Link>
    );
};

const Sidebar = () => {
    const { playlists } = usePlaylists();
    const bg = useColorModeValue('gray.50', 'gray.900');
    const color = useColorModeValue('gray.600', 'gray.400');
    const borderColor = useColorModeValue('gray.200', 'gray.800');
    const logoColor = useColorModeValue('blue.600', 'white');

    return (
        <Box
            w="250px"
            h="100%"
            bg={bg}
            color={color}
            borderRight="1px solid"
            borderColor={borderColor}
            py={5}
            overflowY="auto"
        >
            <Box px={5} mb={8}>
                <Text fontSize="2xl" fontWeight="bold" color={logoColor} letterSpacing="tight">
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
                {/* Admin link */}
                {useAuth().user?.is_admin && (
                    <NavItem icon={FiDisc} to="/admin/users">Admin Users</NavItem>
                )}
            </VStack>
        </Box>
    );
};

export default Sidebar;
