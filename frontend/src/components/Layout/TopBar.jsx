import React from 'react';
import { Flex, IconButton, Input, InputGroup, InputLeftElement, Avatar, Text, useColorMode, Menu, MenuButton, MenuList, MenuItem } from '@chakra-ui/react';
import { FiSearch, FiSun, FiMoon, FiBell, FiUser } from 'react-icons/fi';
import { useAuth } from '../../context/AuthContext';

const TopBar = () => {
    const { colorMode, toggleColorMode } = useColorMode();
    const { user, logout, token } = useAuth();
    const [searchQuery, setSearchQuery] = React.useState('');

    const handleSearch = async (e) => {
        if (e.key === 'Enter' && searchQuery.trim()) {
            // For now, we'll just log or navigate to a search page if we had one
            // Let's at least show we are doing something
            console.log('Searching for:', searchQuery);
            // Redirect to library with search filter?
            // window.location.href = `/library?search=${searchQuery}`;
        }
    };

    return (
        <Flex
            as="header"
            h="60px"
            bg="gray.900"
            borderBottom="1px solid"
            borderColor="gray.800"
            align="center"
            px={6}
            justify="space-between"
        >
            <InputGroup w="400px" size="sm">
                <InputLeftElement pointerEvents="none">
                    <FiSearch color="gray.500" />
                </InputLeftElement>
                <Input
                    type="text"
                    placeholder="Search for songs, artists, albums..."
                    bg="gray.800"
                    border="none"
                    color="white"
                    _focus={{ bg: 'gray.700' }}
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    onKeyDown={handleSearch}
                />
            </InputGroup>

            <Flex align="center" gap={4}>
                <IconButton
                    icon={colorMode === 'light' ? <FiMoon /> : <FiSun />}
                    onClick={toggleColorMode}
                    variant="ghost"
                    aria-label="Toggle theme"
                    color="gray.400"
                />
                <IconButton
                    icon={<FiBell />}
                    variant="ghost"
                    aria-label="Notifications"
                    color="gray.400"
                />

                <Menu>
                    <MenuButton>
                        <Flex align="center" cursor="pointer">
                            <Avatar size="sm" bg="blue.500" icon={<FiUser />} mr={2} name={user?.username} />
                            <Text fontSize="sm" fontWeight="bold" color="white">{user?.username || 'Guest'}</Text>
                        </Flex>
                    </MenuButton>
                    <MenuList bg="gray.800" borderColor="gray.700">
                        <MenuItem bg="gray.800" _hover={{ bg: 'gray.700' }}>Profile</MenuItem>
                        <MenuItem bg="gray.800" _hover={{ bg: 'gray.700' }}>Settings</MenuItem>
                        <MenuItem bg="gray.800" _hover={{ bg: 'gray.700' }} onClick={logout}>Logout</MenuItem>
                    </MenuList>
                </Menu>
            </Flex>
        </Flex>
    );
};

export default TopBar;
