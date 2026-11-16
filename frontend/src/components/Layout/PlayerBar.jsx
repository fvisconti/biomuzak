import React, { useState } from 'react';
import {
    Box, HStack, Text, IconButton, Slider, SliderTrack, SliderFilledTrack, SliderThumb,
    VStack, Button, useColorModeValue
} from '@chakra-ui/react';
import { FiPlay, FiPause, FiSkipBack, FiSkipForward, FiMusic } from 'react-icons/fi';
import { usePlayer } from '../../context/PlayerContext';

const PlayerBar = () => {
    const { currentSong, isPlaying, togglePlay, playNext, playPrev, progress, duration, seek, similarSongs, playSong } = usePlayer();
    const bg = useColorModeValue('white', 'gray.900');
    const border = useColorModeValue('gray.200', 'gray.700');
    const panelBg = useColorModeValue('gray.50', 'gray.800');
    const rowHover = useColorModeValue('gray.100', 'gray.700');
    const [showSimilar, setShowSimilar] = useState(false);

    if (!currentSong) return null;

    return (
        <Box
            position="fixed"
            bottom="0"
            left="0"
            right="0"
            bg={bg}
            borderTop="1px"
            borderColor={border}
            p={4}
            zIndex="1000"
            boxShadow="0 -2px 10px rgba(0,0,0,0.1)"
        >
            {/* Similar tracks panel (click-to-play). Empty state when the
                current song has no embedding or no matches. */}
            {showSimilar && (
                <Box
                    mb={3}
                    p={3}
                    borderRadius="md"
                    bg={panelBg}
                    border="1px"
                    borderColor={border}
                >
                    <Text fontSize="sm" fontWeight="bold" mb={2}>Similar tracks</Text>
                    {similarSongs.length === 0 ? (
                        <Text fontSize="sm" color="gray.500">
                            No similar tracks available for this song.
                        </Text>
                    ) : (
                        <VStack align="stretch" spacing={1}>
                            {similarSongs.map((s) => (
                                <HStack
                                    key={s.id}
                                    spacing={3}
                                    align="center"
                                    cursor="pointer"
                                    px={2}
                                    py={1}
                                    borderRadius="sm"
                                    _hover={{ bg: rowHover }}
                                    onClick={() => playSong(s)}
                                >
                                    <FiMusic color="currentColor" />
                                    <Box flex="1">
                                        <Text fontSize="sm" noOfLines={1}>{s.title}</Text>
                                        <Text fontSize="xs" color="gray.500" noOfLines={1}>{s.artist}</Text>
                                    </Box>
                                    <Text fontSize="xs" color="gray.500">
                                        {Math.round((s.similarity ?? 0) * 100)}%
                                    </Text>
                                </HStack>
                            ))}
                        </VStack>
                    )}
                </Box>
            )}

            <HStack spacing={4} justify="space-between" align="center">
                {/* Song Info */}
                <HStack flex="1" spacing={4}>
                    <Box
                        w="50px"
                        h="50px"
                        bg="gray.300"
                        borderRadius="md"
                        overflow="hidden"
                    >
                        {/* Placeholder for album art if available */}
                        <Box w="100%" h="100%" bg="brand.500" />
                    </Box>
                    <VStack align="start" spacing={0}>
                        <Text fontWeight="bold" noOfLines={1}>{currentSong.title}</Text>
                        <Text fontSize="sm" color="gray.500" noOfLines={1}>{currentSong.artist}</Text>
                    </VStack>
                </HStack>

                {/* Controls */}
                <VStack flex="2" spacing={2}>
                    <HStack spacing={6}>
                        <IconButton
                            icon={<FiSkipBack />}
                            variant="ghost"
                            onClick={playPrev}
                            aria-label="Previous"
                        />
                        <IconButton
                            icon={isPlaying ? <FiPause /> : <FiPlay />}
                            colorScheme="brand"
                            size="lg"
                            isRound
                            onClick={togglePlay}
                            aria-label="Play/Pause"
                        />
                        <IconButton
                            icon={<FiSkipForward />}
                            variant="ghost"
                            onClick={playNext}
                            aria-label="Next"
                        />
                    </HStack>
                    <HStack w="100%" spacing={4}>
                        <Text fontSize="xs">{formatTime(progress)}</Text>
                        <Slider
                            aria-label="progress"
                            value={progress}
                            max={duration || 100}
                            onChange={seek}
                            focusThumbOnChange={false}
                        >
                            <SliderTrack>
                                <SliderFilledTrack bg="brand.500" />
                            </SliderTrack>
                            <SliderThumb />
                        </Slider>
                        <Text fontSize="xs">{formatTime(duration)}</Text>
                    </HStack>
                </VStack>

                {/* Similar tracks toggle */}
                <Box flex="1" display="flex" justifyContent="flex-end">
                    <Button
                        size="sm"
                        variant={showSimilar ? 'solid' : 'outline'}
                        colorScheme="brand"
                        leftIcon={<FiMusic />}
                        onClick={() => setShowSimilar((v) => !v)}
                    >
                        Similar
                    </Button>
                </Box>
            </HStack>
        </Box>
    );
};

const formatTime = (seconds) => {
    if (!seconds) return "0:00";
    const mins = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    return `${mins}:${secs.toString().padStart(2, '0')}`;
};

export default PlayerBar;
