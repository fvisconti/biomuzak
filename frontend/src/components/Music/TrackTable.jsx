import React, { useState } from 'react';
import {
    Table,
    Thead,
    Tbody,
    Tr,
    Th,
    Td,
    IconButton,
    Menu,
    MenuButton,
    MenuList,
    MenuItem,
    Flex,
    Text,
} from '@chakra-ui/react';
import { FiPlus, FiMoreHorizontal, FiPlay } from 'react-icons/fi';
import { useDraggable } from '@dnd-kit/core';
import { CSS } from '@dnd-kit/utilities';

const DraggableTrackRow = ({ track, index }) => {
    const [isHovered, setIsHovered] = useState(false);
    const { attributes, listeners, setNodeRef, transform } = useDraggable({
        id: `track-${track.id}`,
        data: track,
    });

    const style = {
        transform: CSS.Translate.toString(transform),
    };

    return (
        <Tr
            ref={setNodeRef}
            style={style}
            {...listeners}
            {...attributes}
            onMouseEnter={() => setIsHovered(true)}
            onMouseLeave={() => setIsHovered(false)}
            _hover={{ bg: 'whiteAlpha.100', cursor: 'grab' }}
            bg={transform ? 'gray.700' : 'transparent'}
            zIndex={transform ? 999 : 'auto'}
        >
            <Td w="50px">
                {isHovered ? (
                    <IconButton
                        icon={<FiPlay />}
                        variant="ghost"
                        size="sm"
                        aria-label="Play"
                        color="white"
                        onClick={(e) => e.stopPropagation()}
                    />
                ) : (
                    <Text color="gray.500">{index + 1}</Text>
                )}
            </Td>
            <Td>
                <Text fontWeight="bold" color="white">{track.title}</Text>
            </Td>
            <Td color="gray.400">{track.artist}</Td>
            <Td color="gray.400">{track.album}</Td>
            <Td color="gray.400" isNumeric>{track.duration}</Td>
            <Td w="100px">
                <Flex opacity={isHovered ? 1 : 0} transition="opacity 0.2s">
                    <IconButton
                        icon={<FiPlus />}
                        variant="ghost"
                        size="sm"
                        aria-label="Add to Playlist"
                        mr={2}
                        onPointerDown={(e) => e.stopPropagation()} // Prevent drag when clicking action
                    />
                    <Menu>
                        <MenuButton
                            as={IconButton}
                            icon={<FiMoreHorizontal />}
                            variant="ghost"
                            size="sm"
                            aria-label="Options"
                            onPointerDown={(e) => e.stopPropagation()} // Prevent drag
                        />
                        <MenuList bg="gray.800" borderColor="gray.700">
                            <MenuItem bg="gray.800" _hover={{ bg: 'gray.700' }}>Add to Queue</MenuItem>
                            <MenuItem bg="gray.800" _hover={{ bg: 'gray.700' }}>Go to Artist</MenuItem>
                        </MenuList>
                    </Menu>
                </Flex>
            </Td>
        </Tr>
    );
};

const TrackTable = ({ tracks }) => {
    return (
        <Table variant="simple" size="sm">
            <Thead>
                <Tr>
                    <Th w="50px">#</Th>
                    <Th>Title</Th>
                    <Th>Artist</Th>
                    <Th>Album</Th>
                    <Th isNumeric>Time</Th>
                    <Th w="100px"></Th>
                </Tr>
            </Thead>
            <Tbody>
                {tracks.map((track, i) => (
                    <DraggableTrackRow key={track.id} track={track} index={i} />
                ))}
            </Tbody>
        </Table>
    );
};

export default TrackTable;
