import React from 'react';
import { Box, Flex, useToast, useColorModeValue } from '@chakra-ui/react';
import Sidebar from './Sidebar';
import TopBar from './TopBar';
import PlayerBar from './PlayerBar';
import { DndContext, PointerSensor, KeyboardSensor, useSensor, useSensors } from '@dnd-kit/core';
import { sortableKeyboardCoordinates } from '@dnd-kit/sortable';
import { useAuth } from '../../context/AuthContext';
import { useDrag } from '../../context/DragContext';

const MainLayout = ({ children }) => {
    const { token } = useAuth();
    const toast = useToast();
    const { onDragEndHandler } = useDrag();

    const sensors = useSensors(
        useSensor(PointerSensor, {
            activationConstraint: {
                distance: 8,
            },
        }),
        useSensor(KeyboardSensor, {
            coordinateGetter: sortableKeyboardCoordinates,
        })
    );

    const handleDragEnd = async (event) => {
        const { active, over } = event;
        if (!over) return;

        // Check if we dropped on a sidebar playlist
        const overData = over.data.current;

        if (overData && overData.type === 'playlist') {
            const activeData = active.data.current;
            // Support both direct ID and track-ID format
            const songID = activeData.id || (typeof active.id === 'string' && active.id.startsWith('track-') ? active.id.replace('track-', '') : active.id);

            try {
                const res = await fetch(`/api/playlists/${overData.id}/songs`, {
                    method: 'POST',
                    headers: {
                        'Authorization': `Bearer ${token}`,
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({ song_id: parseInt(songID) })
                });

                if (res.ok) {
                    toast({
                        title: "Added to Playlist",
                        description: `Added to ${overData.name}`,
                        status: "success",
                        duration: 2000,
                    });
                }
            } catch (error) {
                console.error("Failed to add song to playlist via drag", error);
            }
            return;
        }

        // Delegate to child component handler if available (e.g. for reordering)
        if (onDragEndHandler.current) {
            onDragEndHandler.current(event);
        }
    };

    return (
        <Box h="100vh" bg={useColorModeValue('white', '#0d0d0d')} overflow="hidden">
            <DndContext sensors={sensors} onDragEnd={handleDragEnd}>
                <Flex h="calc(100vh - 90px)">
                    <Sidebar />
                    <Box flex="1" display="flex" flexDirection="column">
                        <TopBar />
                        <Box flex="1" overflowY="auto" p={6}>
                            {children}
                        </Box>
                    </Box>
                </Flex>
            </DndContext>
            <PlayerBar />
        </Box>
    );
};

export default MainLayout;
