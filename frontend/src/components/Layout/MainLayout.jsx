import React from 'react';
import { Box, Flex } from '@chakra-ui/react';
import Sidebar from './Sidebar';
import TopBar from './TopBar';
import PlayerBar from './PlayerBar';

const MainLayout = ({ children }) => {
    return (
        <Box h="100vh" bg="#0d0d0d" overflow="hidden">
            <Flex h="calc(100vh - 90px)">
                <Sidebar />
                <Box flex="1" display="flex" flexDirection="column">
                    <TopBar />
                    <Box flex="1" overflowY="auto" p={6}>
                        {children}
                    </Box>
                </Box>
            </Flex>
            <PlayerBar />
        </Box>
    );
};

export default MainLayout;
