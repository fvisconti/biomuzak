import React from 'react';
import { Tr, Td, IconButton, Icon, Text, HStack } from '@chakra-ui/react';
import { FiPlay, FiMusic, FiTrash2, FiDownload } from 'react-icons/fi';
import { useSortable } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';

const SortableSongRow = ({ song, index, token, handlePlaySong, handleDeleteSong }) => {
    const {
        attributes,
        listeners,
        setNodeRef,
        transform,
        transition,
    } = useSortable({
        id: song.id,
        data: { id: song.id, ...song }
    });

    const style = {
        transform: CSS.Transform.toString(transform),
        transition,
    };

    return (
        <Tr ref={setNodeRef} style={style} {...attributes} {...listeners} _hover={{ bg: 'whiteAlpha.50' }}>
            <Td>
                <IconButton
                    icon={<FiPlay />}
                    size="xs"
                    variant="ghost"
                    colorScheme="brand"
                    onClick={(e) => { e.stopPropagation(); handlePlaySong(index); }}
                    aria-label="Play"
                    onPointerDown={(e) => e.stopPropagation()}
                />
            </Td>
            <Td fontWeight="bold">
                <HStack>
                    <Icon as={FiMusic} color="brand.400" />
                    <Text>{song.title || 'Unknown Title'}</Text>
                </HStack>
            </Td>
            <Td>{song.artist || '-'}</Td>
            <Td>{song.album || '-'}</Td>
            <Td fontFamily="monospace">
                {Math.floor(song.duration / 60)}:{(song.duration % 60).toString().padStart(2, '0')}
            </Td>
            <Td>
                <HStack spacing={1}>
                    <IconButton
                        icon={<FiDownload />}
                        size="sm"
                        variant="ghost"
                        colorScheme="blue"
                        onClick={(e) => {
                            e.stopPropagation();
                            const url = `/api/songs/${song.id}/download?token=${token}`;
                            window.open(url, '_blank');
                        }}
                        aria-label="Download song"
                        onPointerDown={(e) => e.stopPropagation()}
                    />
                    <IconButton
                        icon={<FiTrash2 />}
                        size="sm"
                        colorScheme="red"
                        variant="ghost"
                        onClick={(e) => { e.stopPropagation(); handleDeleteSong(song.id); }}
                        aria-label="Remove song"
                        onPointerDown={(e) => e.stopPropagation()}
                    />
                </HStack>
            </Td>
        </Tr>
    );
};

export default SortableSongRow;
