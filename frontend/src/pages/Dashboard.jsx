import React, { useState } from 'react';
import { Box, Heading, Text, SimpleGrid, Flex, Image, useToast } from '@chakra-ui/react';
import TrackTable from '../components/Music/TrackTable';
import { dummyTracks, dummyPlaylists } from '../data/dummyData';
import { DndContext, useDroppable, DragOverlay } from '@dnd-kit/core';

const DroppablePlaylistCard = ({ playlist }) => {
    const { isOver, setNodeRef } = useDroppable({
        id: `playlist-${playlist.id}`,
        data: playlist,
    });

    return (
        <Box
            ref={setNodeRef}
            p={4}
            bg={isOver ? 'blue.900' : 'gray.800'}
            _hover={{ bg: 'gray.700' }}
            cursor="pointer"
            transition="background 0.2s"
            border={isOver ? '2px solid' : '2px solid transparent'}
            borderColor="blue.400"
            borderRadius="md"
        >
            <Image src={playlist.cover} mb={4} borderRadius="sm" opacity={isOver ? 0.8 : 1} />
            <Text fontWeight="bold" fontSize="lg" mb={1}>{playlist.name}</Text>
            <Text fontSize="sm" color="gray.500">{playlist.count} Tracks</Text>
            {isOver && <Text color="blue.300" fontSize="sm" mt={2}>Drop to add track!</Text>}
        </Box>
    );
};

const Dashboard = () => {
    const toast = useToast();
    const [activeId, setActiveId] = useState(null);

    const handleDragStart = (event) => {
        setActiveId(event.active.id);
    };

    const handleDragEnd = (event) => {
        const { active, over } = event;
        setActiveId(null);

        if (over && active) {
            const trackTitle = active.data.current?.title || 'Track';
            const playlistName = over.data.current?.name || 'Playlist';

            toast({
                title: "Added to Playlist",
                description: `Added "${trackTitle}" to "${playlistName}"`,
                status: "success",
                duration: 3000,
                isClosable: true,
                position: 'bottom-right',
            });
        }
    };

    return (
        <DndContext onDragStart={handleDragStart} onDragEnd={handleDragEnd}>
            <Box>
                <Heading mb={6} fontSize="3xl">Music Center</Heading>

                <Box mb={10}>
                    <Text fontSize="xl" fontWeight="bold" mb={4} textTransform="uppercase" letterSpacing="wide" color="gray.400">
                        Your Playlists
                    </Text>
                    <SimpleGrid columns={{ base: 2, md: 3, lg: 4, xl: 5 }} spacing={6}>
                        {dummyPlaylists.map((pl) => (
                            <DroppablePlaylistCard key={pl.id} playlist={pl} />
                        ))}
                        {/* Add New Playlist Placeholder */}
                        <Flex
                            bg="gray.800"
                            border="2px dashed"
                            borderColor="gray.600"
                            align="center"
                            justify="center"
                            direction="column"
                            cursor="pointer"
                            _hover={{ borderColor: 'white', color: 'white' }}
                            color="gray.500"
                            h="100%"
                            minH="250px"
                            borderRadius="md"
                        >
                            <Text fontSize="5xl" fontWeight="light">+</Text>
                            <Text mt={2}>Create Playlist</Text>
                        </Flex>
                    </SimpleGrid>
                </Box>

                <Box>
                    <Text fontSize="xl" fontWeight="bold" mb={4} textTransform="uppercase" letterSpacing="wide" color="gray.400">
                        Suggested For You
                    </Text>
                    <Text fontSize="sm" color="gray.500" mb={4}>Drag tracks to playlists above to add them.</Text>
                    <TrackTable tracks={dummyTracks} />
                </Box>
            </Box>
            <DragOverlay>
                {activeId ? (
                    <Box p={2} bg="gray.700" color="white" borderRadius="md" boxShadow="lg">
                        Dragging Track...
                    </Box>
                ) : null}
            </DragOverlay>
        </DndContext>
    );
};

export default Dashboard;
