import React from 'react';
import { Flex, IconButton, Input, InputGroup, InputLeftElement, Avatar, Text, useColorMode, Menu, MenuButton, MenuList, MenuItem, useColorModeValue } from '@chakra-ui/react';
import { FiSearch, FiSun, FiMoon, FiBell, FiUser } from 'react-icons/fi';
import { useAuth } from '../../context/AuthContext';

const TopBar = () => {
    const { colorMode, toggleColorMode } = useColorMode();
    const { user, logout, token } = useAuth();
    const [searchQuery, setSearchQuery] = React.useState('');

    const bg = useColorModeValue('white', 'gray.900');
    const borderColor = useColorModeValue('gray.200', 'gray.800');
    const searchBg = useColorModeValue('gray.100', 'gray.800');
    const textColor = useColorModeValue('gray.800', 'white');
    const menuBg = useColorModeValue('white', 'gray.800');
    const menuHoverBg = useColorModeValue('gray.100', 'gray.700');

    const handleSearch = async (e) => {
        if (e.key === 'Enter' && searchQuery.trim()) {
            console.log('Searching for:', searchQuery);
            window.location.href = `/library?q=${encodeURIComponent(searchQuery)}`;
        }
    };

    return (
        <Flex
            as="header"
            h="60px"
            bg={bg}
            borderBottom="1px solid"
            borderColor={borderColor}
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
                    bg={searchBg}
                    border="none"
                    color={textColor}
                    _focus={{ bg: useColorModeValue('gray.200', 'gray.700') }}
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
                            <Text fontSize="sm" fontWeight="bold" color={textColor}>{user?.username || 'Guest'}</Text>
                        </Flex>
                    </MenuButton>
                    <MenuList bg={menuBg} borderColor={borderColor}>
                        <MenuItem bg={menuBg} _hover={{ bg: menuHoverBg }}>Profile</MenuItem>
                        <MenuItem bg={menuBg} _hover={{ bg: menuHoverBg }}>Settings</MenuItem>
                        <MenuItem bg={menuBg} _hover={{ bg: menuHoverBg }} onClick={logout}>Logout</MenuItem>
                    </MenuList>
                </Menu>
            </Flex>
        </Flex>
    );
};

export default TopBar;
