import React from 'react';
import { Box, Heading, Text, VStack, Container } from '@chakra-ui/react';
import FileUploader from '../components/Upload/FileUploader';

const Upload = () => {
    return (
        <Container maxW="container.lg" py={8}>
            <VStack spacing={8} align="stretch">
                <Box>
                    <Heading as="h1" size="xl" mb={2}>
                        Upload Music
                    </Heading>
                    <Text color="gray.400">
                        Add new music to your library. Supported formats: MP3, FLAC, WAV, M4A, OGG.
                    </Text>
                </Box>

                <FileUploader />
            </VStack>
        </Container>
    );
};

export default Upload;
