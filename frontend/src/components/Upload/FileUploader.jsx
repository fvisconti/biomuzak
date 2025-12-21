import React, { useCallback, useState, useEffect, useRef } from 'react';
import { useDropzone } from 'react-dropzone';
import { useLocation } from 'react-router-dom';
import {
    Box,
    Button,
    Center,
    VStack,
    Text,
    List,
    ListItem,
    ListIcon,
    Progress,
    useToast,
    Icon,
    HStack,
    Select
} from '@chakra-ui/react';
import { FiUploadCloud, FiFile, FiCheck, FiFolder } from 'react-icons/fi';
import { useAuth } from '../../context/AuthContext';

// Helper to recursively read entries
const scanFiles = async (item) => {
    if (item.isFile) {
        return new Promise((resolve) => {
            item.file((file) => {
                resolve([file]);
            });
        });
    } else if (item.isDirectory) {
        const directoryReader = item.createReader();
        const entries = await new Promise((resolve) => {
            directoryReader.readEntries((entries) => {
                resolve(entries);
            });
        });
        const files = await Promise.all(entries.map(scanFiles));
        return files.flat();
    }
    return [];
};

const FileUploader = () => {
    const location = useLocation();
    const [queue, setQueue] = useState([]);
    const [completed, setCompleted] = useState([]);
    const [uploading, setUploading] = useState(false);
    const [currentFile, setCurrentFile] = useState(null);
    const [progress, setProgress] = useState(0);
    const [playlists, setPlaylists] = useState([]);
    const [selectedPlaylist, setSelectedPlaylist] = useState(location.state?.playlistID || '');
    const [newPlaylistName, setNewPlaylistName] = useState('');

    const { token } = useAuth();
    const toast = useToast();
    const folderInputRef = useRef(null);

    // Fetch playlists for dropdown
    useEffect(() => {
        const fetchPlaylists = async () => {
            try {
                const res = await fetch('/api/playlists', {
                    headers: { 'Authorization': `Bearer ${token}` }
                });
                if (res.ok) {
                    const data = await res.json();
                    setPlaylists(data || []);
                }
            } catch (err) {
                console.error("Failed to fetch playlists", err);
            }
        };
        fetchPlaylists();
    }, [token]);

    // Update selected playlist if passed via navigation state (e.g. from PlaylistDetails)
    useEffect(() => {
        if (location.state?.playlistID) {
            setSelectedPlaylist(location.state.playlistID);
        }
    }, [location.state]);

    // Process the queue
    useEffect(() => {
        if (uploading && queue.length > 0 && !currentFile) {
            uploadNextFile();
        } else if (uploading && queue.length === 0 && !currentFile) {
            setUploading(false);
            toast({
                title: 'All uploads completed',
                status: 'success',
                duration: 3000,
                isClosable: true,
            });
        }
    }, [uploading, queue, currentFile]);

    const uploadNextFile = async () => {
        const fileToUpload = queue[0];
        setCurrentFile(fileToUpload);
        setQueue(prev => prev.slice(1));
        setProgress(0);

        const formData = new FormData();
        formData.append('files', fileToUpload);
        if (selectedPlaylist && selectedPlaylist !== 'NEW') {
            formData.append('playlist_id', selectedPlaylist);
        } else if (newPlaylistName) {
            formData.append('new_playlist_name', newPlaylistName);
        }

        try {
            const xhr = new XMLHttpRequest();
            xhr.open('POST', '/api/upload');
            xhr.setRequestHeader('Authorization', `Bearer ${token}`);

            xhr.upload.onprogress = (event) => {
                if (event.lengthComputable) {
                    const percentComplete = (event.loaded / event.total) * 100;
                    setProgress(percentComplete);
                }
            };

            xhr.onload = () => {
                if (xhr.status === 200) {
                    setCompleted(prev => [...prev, fileToUpload.name]);
                } else {
                    console.error('Upload failed for', fileToUpload.name);
                    toast({
                        title: `Failed: ${fileToUpload.name}`,
                        status: 'error',
                        duration: 3000,
                        isClosable: true,
                    });
                }
                setCurrentFile(null);
            };

            xhr.onerror = () => {
                console.error('Network error for', fileToUpload.name);
                setCurrentFile(null);
            };

            xhr.send(formData);
        } catch (error) {
            console.error('Upload error:', error);
            setCurrentFile(null);
        }
    };

    const handleFilesAdded = (newFiles) => {
        const audioFiles = newFiles.filter(file =>
            file.name.match(/\.(mp3|flac|wav|m4a|ogg)$/i) && !file.name.startsWith('._')
        );

        if (audioFiles.length > 0) {
            setQueue(prev => [...prev, ...audioFiles]);
            toast({
                title: `Added ${audioFiles.length} files to queue`,
                status: "info",
                duration: 2000,
            });
        }
    };

    const onDrop = useCallback(async (acceptedFiles, fileRejections, event) => {
        let newFiles = [];
        const items = event.dataTransfer ? event.dataTransfer.items : [];

        // Detect folder name if a single folder was dropped
        if (items && items.length === 1 && items[0].webkitGetAsEntry().isDirectory) {
            const entry = items[0].webkitGetAsEntry();
            setNewPlaylistName(entry.name);
            setSelectedPlaylist('NEW');
        }

        if (items && items.length > 0 && items[0].webkitGetAsEntry) {
            const entries = [];
            for (let i = 0; i < items.length; i++) {
                entries.push(items[i].webkitGetAsEntry());
            }
            const results = await Promise.all(entries.map(scanFiles));
            newFiles = results.flat();
        } else {
            newFiles = acceptedFiles;
        }
        handleFilesAdded(newFiles);
    }, [toast]);

    const { getRootProps, getInputProps, isDragActive } = useDropzone({
        onDrop,
        noClick: false,
        noKeyboard: true
    });

    // Handle manual folder selection via the hidden input
    const handleFolderSelect = (e) => {
        if (e.target.files && e.target.files.length > 0) {
            // Try to get folder name from the first file's path
            const firstFile = e.target.files[0];
            const relativePath = firstFile.webkitRelativePath;
            if (relativePath) {
                const folderName = relativePath.split('/')[0];
                if (folderName) {
                    setNewPlaylistName(folderName);
                    setSelectedPlaylist('NEW');
                }
            }
            handleFilesAdded(Array.from(e.target.files));
        }
    };

    const startUpload = () => {
        if (queue.length > 0) {
            setUploading(true);
        }
    };

    const stopUpload = () => {
        setUploading(false);
    };

    const clearCompleted = () => {
        setCompleted([]);
    };

    return (
        <VStack spacing={6} align="stretch" w="full" maxW="800px" mx="auto">
            {/* Playlist Selection */}
            <Box>
                <Text mb={2} fontWeight="bold">Add to Playlist (Optional)</Text>
                <HStack spacing={4}>
                    <Select
                        placeholder="Select playlist..."
                        value={selectedPlaylist}
                        onChange={(e) => {
                            setSelectedPlaylist(e.target.value);
                            if (e.target.value !== 'NEW') setNewPlaylistName('');
                        }}
                        bg="gray.700"
                        flex="1"
                    >
                        {playlists.map(pl => (
                            <option key={pl.id} value={pl.id}>{pl.name}</option>
                        ))}
                        <option value="NEW">+ Create New Playlist</option>
                    </Select>

                    {selectedPlaylist === 'NEW' && (
                        <Box flex="1">
                            <input
                                placeholder="Enter playlist name..."
                                value={newPlaylistName}
                                onChange={(e) => setNewPlaylistName(e.target.value)}
                                style={{
                                    width: '100%',
                                    padding: '8px 12px',
                                    backgroundColor: '#2D3748',
                                    borderRadius: '6px',
                                    border: '1px solid #4A5568',
                                    color: 'white'
                                }}
                            />
                        </Box>
                    )}
                </HStack>
            </Box>

            {/* Dropzone */}
            <Box
                {...getRootProps()}
                p={10}
                border="2px dashed"
                borderColor={isDragActive ? 'brand.500' : 'gray.600'}
                borderRadius="md"
                bg={isDragActive ? 'whiteAlpha.100' : 'transparent'}
                cursor="pointer"
                transition="all 0.2s"
                _hover={{
                    borderColor: 'brand.500',
                    bg: 'whiteAlpha.50',
                }}
            >
                <input {...getInputProps()} />
                <Center flexDirection="column">
                    <Icon as={FiUploadCloud} w={10} h={10} mb={4} color="brand.400" />
                    {isDragActive ? (
                        <Text>Drop files or folders here ...</Text>
                    ) : (
                        <Text>Drag & drop audio files or folders, or click to select files</Text>
                    )}
                    <Text fontSize="sm" color="gray.500" mt={2}>
                        Supports MP3, FLAC, WAV, M4A, OGG
                    </Text>
                </Center>
            </Box>

            {/* Manual Buttons */}
            <HStack justify="center">
                <Button onClick={() => document.querySelector('input[type="file"]').click()}>
                    Select Files
                </Button>
                <Button onClick={() => folderInputRef.current.click()} leftIcon={<FiFolder />}>
                    Select Folder
                </Button>
                {/* Hidden folder input */}
                <input
                    type="file"
                    ref={folderInputRef}
                    onChange={handleFolderSelect}
                    webkitdirectory=""
                    directory=""
                    multiple
                    style={{ display: 'none' }}
                />
            </HStack>

            {/* Queue UI */}
            <Box>
                <HStack justify="space-between" mb={2}>
                    <Text fontWeight="bold">Queue ({queue.length})</Text>
                    <HStack>
                        {uploading ? (
                            <Button size="sm" colorScheme="red" onClick={stopUpload}>Pause/Stop</Button>
                        ) : (
                            <Button
                                size="sm"
                                colorScheme="brand"
                                onClick={startUpload}
                                isDisabled={queue.length === 0}
                            >
                                Start Upload
                            </Button>
                        )}
                    </HStack>
                </HStack>

                {currentFile && (
                    <Box mb={4} p={4} bg="gray.800" borderRadius="md" borderWidth="1px" borderColor="brand.500">
                        <HStack justify="space-between" mb={2}>
                            <Text isTruncated>Uploading: <b>{currentFile.name}</b></Text>
                            <Text fontSize="sm">{Math.round(progress)}%</Text>
                        </HStack>
                        <Progress value={progress} size="sm" colorScheme="brand" hasStripe isAnimated />
                    </Box>
                )}

                {queue.length > 0 && !uploading && (
                    <List spacing={2} maxH="150px" overflowY="auto" pr={2} mb={4}>
                        {queue.map((file, index) => (
                            <ListItem key={index} display="flex" alignItems="center" fontSize="sm" color="gray.400">
                                <ListIcon as={FiFile} />
                                <Text isTruncated>{file.name}</Text>
                            </ListItem>
                        ))}
                    </List>
                )}
            </Box>

            {completed.length > 0 && (
                <Box>
                    <HStack justify="space-between" mb={2}>
                        <Text fontWeight="bold" color="green.400">Completed ({completed.length})</Text>
                        <Button size="xs" variant="ghost" onClick={clearCompleted}>Clear</Button>
                    </HStack>
                    <List spacing={2} maxH="200px" overflowY="auto" pr={2}>
                        {completed.map((name, index) => (
                            <ListItem key={index} display="flex" alignItems="center" color="green.300" fontSize="sm">
                                <ListIcon as={FiCheck} />
                                <Text isTruncated>{name}</Text>
                            </ListItem>
                        ))}
                    </List>
                </Box>
            )}
        </VStack>
    );
};

export default FileUploader;
