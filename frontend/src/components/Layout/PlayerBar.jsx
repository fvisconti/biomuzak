import React, { useState } from 'react';
import { Box, Flex, Text, IconButton, Slider, SliderTrack, SliderFilledTrack, SliderThumb, Icon, Image } from '@chakra-ui/react';
import { FiPlay, FiPause, FiSkipBack, FiSkipForward, FiRepeat, FiShuffle, FiVolume2 } from 'react-icons/fi';

const PlayerBar = () => {
    const [isPlaying, setIsPlaying] = useState(false);

    return (
        <Box
            h="90px"
            bg="gray.900"
            borderTop="1px solid"
            borderColor="gray.800"
            px={6}
            position="fixed"
            bottom={0}
            left={0}
            right={0}
            zIndex={100}
        >
            <Flex h="100%" align="center" justify="space-between">
                {/* Track Info */}
                <Flex w="30%" align="center" gap={4}>
                    <Box w="56px" h="56px" bg="gray.800" borderRadius="md" overflow="hidden">
                        {/* Placeholder for Album Art */}
                        <Image src="https://via.placeholder.com/56" alt="Album Art" />
                    </Box>
                    <Box>
                        <Text fontWeight="bold" color="white" fontSize="sm">Track Title</Text>
                        <Text fontSize="xs" color="gray.400">Artist Name</Text>
                    </Box>
                </Flex>

                {/* Controls */}
                <Flex direction="column" align="center" w="40%">
                    <Flex align="center" gap={6} mb={2}>
                        <IconButton icon={<FiShuffle />} variant="ghost" size="sm" color="gray.400" aria-label="Shuffle" />
                        <IconButton icon={<FiSkipBack />} variant="ghost" size="sm" color="gray.300" aria-label="Previous" />
                        <IconButton
                            icon={isPlaying ? <FiPause fill="currentColor" /> : <FiPlay fill="currentColor" />}
                            onClick={() => setIsPlaying(!isPlaying)}
                            variant="solid"
                            colorScheme="whiteAlpha"
                            bg="white"
                            color="black"
                            rounded="full"
                            size="md"
                            aria-label="Play/Pause"
                            _hover={{ bg: 'gray.200' }}
                        />
                        <IconButton icon={<FiSkipForward />} variant="ghost" size="sm" color="gray.300" aria-label="Next" />
                        <IconButton icon={<FiRepeat />} variant="ghost" size="sm" color="gray.400" aria-label="Repeat" />
                    </Flex>
                    <Flex w="100%" align="center" gap={3}>
                        <Text fontSize="xs" color="gray.500">0:00</Text>
                        <Slider aria-label="progress" defaultValue={30} size="sm">
                            <SliderTrack bg="gray.700">
                                <SliderFilledTrack bg="white" />
                            </SliderTrack>
                            <SliderThumb boxSize={2} />
                        </Slider>
                        <Text fontSize="xs" color="gray.500">3:45</Text>
                    </Flex>
                </Flex>

                {/* Volume */}
                <Flex w="30%" justify="flex-end" align="center" gap={2}>
                    <Icon as={FiVolume2} color="gray.400" />
                    <Slider aria-label="volume" defaultValue={80} w="100px" size="sm">
                        <SliderTrack bg="gray.700">
                            <SliderFilledTrack bg="white" />
                        </SliderTrack>
                        <SliderThumb boxSize={2} />
                    </Slider>
                </Flex>
            </Flex>
        </Box>
    );
};

export default PlayerBar;
