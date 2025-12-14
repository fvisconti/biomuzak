import React from 'react';
import { Box, VStack, Link, Icon, Text, Divider } from '@chakra-ui/react';
import { FiHome, FiMusic, FiList, FiDisc, FiRadio } from 'react-icons/fi';
import { Link as RouterLink, useLocation } from 'react-router-dom';

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

const Sidebar = () => {
    return (
        <Box
            w="250px"
            h="100%"
            bg="gray.900"
            color="gray.400"
            borderRight="1px solid"
            borderColor="gray.800"
            py={5}
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
                <NavItem icon={FiRadio} to="/radio">Radio</NavItem>

                <Divider my={4} borderColor="gray.700" />

                <Text px={5} fontSize="xs" fontWeight="bold" textTransform="uppercase" letterSpacing="widest" mb={2}>
                    Your Collection
                </Text>
                <NavItem icon={FiList} to="/playlists">Playlists</NavItem>
                <NavItem icon={FiDisc} to="/albums">Albums</NavItem>
            </VStack>
        </Box>
    );
};

export default Sidebar;
