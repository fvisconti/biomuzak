import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
    Box, Heading, VStack, HStack, Text, Button, Input, Select, Checkbox,
    Tabs, TabList, Tab, TabPanels, TabPanel, Spinner, Badge, Progress,
    useToast, Divider, IconButton, Switch, FormControl, FormLabel,
    FormHelperText, Table, Thead, Tbody, Tr, Th, Td,
    Code, Link as ChakraLink, Flex
} from '@chakra-ui/react';
import {
    FiLink, FiLogOut, FiRefreshCw, FiDownload, FiSettings, FiExternalLink,
    FiMusic, FiDisc, FiList, FiCheck, FiX, FiAlertCircle
} from 'react-icons/fi';
import {
    pairStart, pairStatus, getStatus, saveSettings, disconnect,
    getLibrary, getAlbum, getPlaylist, startImport, importStatus
} from '../api/tidal';

const QUALITIES = [
    { value: 'low', label: 'Low (128kbps, m4a)' },
    { value: 'normal', label: 'Normal (320kbps, m4a)' },
    { value: 'high', label: 'High (Lossless FLAC)' },
    { value: 'max', label: 'Max (Hi-Res FLAC)' },
];

const FILTERS = [
    { value: 'include', label: 'Include' },
    { value: 'exclude', label: 'Exclude' },
    { value: 'none', label: 'No preference' },
];

// ---------------------------------------------------------------------------
// Pairing card (device flow)
// ---------------------------------------------------------------------------
const PairingCard = ({ onConnected }) => {
    const toast = useToast();
    const [pairing, setPairing] = useState(null);
    const [pairState, setPairState] = useState('idle'); // idle | pending | authorized | expired
    const [error, setError] = useState('');
    const pollRef = useRef(null);

    const stopPolling = useCallback(() => {
        if (pollRef.current) {
            clearInterval(pollRef.current);
            pollRef.current = null;
        }
    }, []);

    useEffect(() => () => stopPolling(), [stopPolling]);

    const startPairing = async () => {
        setError('');
        stopPolling();
        try {
            const res = await pairStart();
            const data = res.data;
            setPairing(data);
            setPairState('pending');
            const interval = Math.max(2, data.interval || 2) * 1000;
            pollRef.current = setInterval(async () => {
                try {
                    const s = await pairStatus(data.pairId);
                    const st = s.data.status;
                    if (st === 'authorized') {
                        stopPolling();
                        setPairState('authorized');
                        toast({ title: `Connected as ${s.data.tidalUsername}`, status: 'success', duration: 3000 });
                        onConnected();
                    } else if (st === 'expired') {
                        stopPolling();
                        setPairState('expired');
                        setError('Pairing expired. Please try again.');
                    }
                } catch (e) {
                    // transient network errors are ignored; keep polling
                }
            }, interval);
        } catch (e) {
            setError(e.message || 'Failed to start pairing');
        }
    };

    return (
        <Box
            p={6}
            border="1px solid"
            borderColor="gray.700"
            borderRadius="md"
            bg="gray.800"
        >
            <VStack spacing={4} align="stretch">
                <HStack>
                    <FiLink color="blue.400" />
                    <Heading as="h2" size="md">Connect Tidal</Heading>
                </HStack>
                <Text color="gray.400" fontSize="sm">
                    Link your personal Tidal account to import tracks, albums, and
                    playlists. You will be asked to confirm on Tidal's website.
                </Text>

                {pairState === 'idle' && (
                    <Button colorScheme="blue" onClick={startPairing} leftIcon={<FiLink />}>
                        Connect Tidal
                    </Button>
                )}

                {pairState === 'pending' && pairing && (
                    <VStack spacing={3} align="stretch">
                        <Text fontSize="sm" color="gray.300">
                            Enter this code on{' '}
                            <ChakraLink color="blue.400" href={pairing.verificationUri} isExternal>
                                {pairing.verificationUri}
                            </ChakraLink>
                            :
                        </Text>
                        <Code fontSize="2xl" p={3} textAlign="center" letterSpacing="widest">
                            {pairing.userCode}
                        </Code>
                        <HStack justify="center" spacing={3}>
                            <Button
                                size="sm"
                                variant="outline"
                                leftIcon={<FiExternalLink />}
                                onClick={() => window.open(pairing.verificationUriComplete, '_blank')}
                            >
                                Open Tidal to confirm
                            </Button>
                            <Button size="sm" variant="ghost" onClick={() => { stopPolling(); setPairState('idle'); setPairing(null); }}>
                                Cancel
                            </Button>
                        </HStack>
                        <HStack justify="center" spacing={2}>
                            <Spinner size="sm" color="blue.400" />
                            <Text fontSize="sm" color="gray.400">Waiting for confirmation…</Text>
                        </HStack>
                    </VStack>
                )}

                {pairState === 'authorized' && (
                    <HStack spacing={2} color="green.400">
                        <FiCheck />
                        <Text>Connected successfully.</Text>
                    </HStack>
                )}

                {error && (
                    <HStack spacing={2} color="red.400" fontSize="sm">
                        <FiAlertCircle />
                        <Text>{error}</Text>
                    </HStack>
                )}
            </VStack>
        </Box>
    );
};

// ---------------------------------------------------------------------------
// Settings panel
// ---------------------------------------------------------------------------
const SettingsPanel = ({ config, onSave, saving }) => {
    const [draft, setDraft] = useState(config);
    const [dirty, setDirty] = useState(false);

    useEffect(() => {
        setDraft(config);
        setDirty(false);
    }, [config]);

    const update = (key, value) => {
        setDraft((d) => ({ ...d, [key]: value }));
        setDirty(true);
    };

    const num = (v) => (v === '' ? 0 : Number(v));

    return (
        <Box
            p={5}
            border="1px solid"
            borderColor="gray.700"
            borderRadius="md"
            bg="gray.800"
        >
            <VStack spacing={4} align="stretch">
                <HStack>
                    <FiSettings color="blue.400" />
                    <Heading as="h2" size="md">Import &amp; Export Settings</Heading>
                </HStack>

                <Flex wrap="wrap" gap={5}>
                    <FormControl w="260px">
                        <FormLabel fontSize="sm">Audio quality</FormLabel>
                        <Select
                            value={draft.quality || 'high'}
                            onChange={(e) => update('quality', e.target.value)}
                        >
                            {QUALITIES.map((q) => (
                                <option key={q.value} value={q.value}>{q.label}</option>
                            ))}
                        </Select>
                    </FormControl>

                    <FormControl w="140px">
                        <FormLabel fontSize="sm">Concurrency</FormLabel>
                        <Input
                            type="number"
                            min={1}
                            max={8}
                            value={draft.concurrency ?? 4}
                            onChange={(e) => update('concurrency', num(e.target.value))}
                        />
                        <FormHelperText>1–8 parallel downloads</FormHelperText>
                    </FormControl>

                    <FormControl w="140px">
                        <FormLabel fontSize="sm">Min delay (s)</FormLabel>
                        <Input
                            type="number"
                            min={0}
                            value={draft.delay_min_sec ?? 2}
                            onChange={(e) => update('delay_min_sec', num(e.target.value))}
                        />
                    </FormControl>

                    <FormControl w="140px">
                        <FormLabel fontSize="sm">Max delay (s)</FormLabel>
                        <Input
                            type="number"
                            min={0}
                            value={draft.delay_max_sec ?? 8}
                            onChange={(e) => update('delay_max_sec', num(e.target.value))}
                        />
                        <FormHelperText>Random pause between downloads</FormHelperText>
                    </FormControl>
                </Flex>

                <Flex wrap="wrap" gap={5}>
                    <FormControl w="200px">
                        <FormLabel fontSize="sm">Singles</FormLabel>
                        <Select
                            value={draft.singles_filter || 'none'}
                            onChange={(e) => update('singles_filter', e.target.value)}
                        >
                            {FILTERS.map((f) => (
                                <option key={f.value} value={f.value}>{f.label}</option>
                            ))}
                        </Select>
                    </FormControl>

                    <FormControl w="200px">
                        <FormLabel fontSize="sm">Videos</FormLabel>
                        <Select
                            value={draft.videos_filter || 'none'}
                            onChange={(e) => update('videos_filter', e.target.value)}
                        >
                            {FILTERS.map((f) => (
                                <option key={f.value} value={f.value}>{f.label}</option>
                            ))}
                        </Select>
                    </FormControl>

                    <FormControl w="200px">
                        <FormLabel fontSize="sm">Atmos</FormLabel>
                        <Select
                            value={draft.atmos_filter || 'none'}
                            onChange={(e) => update('atmos_filter', e.target.value)}
                        >
                            {FILTERS.map((f) => (
                                <option key={f.value} value={f.value}>{f.label}</option>
                            ))}
                        </Select>
                    </FormControl>
                </Flex>

                <Divider />

                <Text fontSize="sm" fontWeight="bold" color="gray.300">
                    Export to local folder
                </Text>
                <FormControl>
                    <FormLabel fontSize="sm">Export path (absolute, on the server)</FormLabel>
                    <Input
                        placeholder="/data/exports"
                        value={draft.export_path || ''}
                        onChange={(e) => update('export_path', e.target.value)}
                    />
                    <FormHelperText>
                        Where exported files are written. Leave empty to disable export.
                    </FormHelperText>
                </FormControl>
                <FormControl>
                    <FormLabel fontSize="sm">Folder template</FormLabel>
                    <Input
                        value={draft.export_template || '{artist}/{album}/{title}'}
                        onChange={(e) => update('export_template', e.target.value)}
                    />
                    <FormHelperText>
                        Placeholders: {'{artist} {album} {title} {year} {genre}'}
                    </FormHelperText>
                </FormControl>
                <HStack spacing={3}>
                    <Switch
                        isChecked={draft.export_embed_tags !== false}
                        onChange={(e) => update('export_embed_tags', e.target.checked)}
                    />
                    <Text fontSize="sm">Embed tags into exported FLAC files</Text>
                </HStack>

                <HStack justify="flex-end">
                    <Button
                        colorScheme="blue"
                        size="sm"
                        isDisabled={!dirty || saving}
                        isLoading={saving}
                        onClick={() => onSave(draft)}
                    >
                        Save settings
                    </Button>
                </HStack>
            </VStack>
        </Box>
    );
};

// ---------------------------------------------------------------------------
// Track import table (selection + editable metadata)
// ---------------------------------------------------------------------------
const TrackImportTable = ({ tracks, selected, setSelected }) => {
    const toggle = (track) => {
        setSelected((prev) => {
            const next = { ...prev };
            if (next[track.id]) {
                delete next[track.id];
            } else {
                next[track.id] = {
                    track_id: track.id,
                    title: track.title || '',
                    artist: track.artistName || '',
                    album: track.albumName || '',
                    year: track.year || 0,
                };
            }
            return next;
        });
    };

    const updateField = (trackId, field, value) => {
        setSelected((prev) => {
            if (!prev[trackId]) return prev;
            return { ...prev, [trackId]: { ...prev[trackId], [field]: value } };
        });
    };

    if (!tracks || tracks.length === 0) {
        return (
            <Text color="gray.500" py={6} textAlign="center">
                No tracks found.
            </Text>
        );
    }

    return (
        <Box overflowX="auto">
            <Table variant="simple" size="sm">
                <Thead>
                    <Tr>
                        <Th w="40px"></Th>
                        <Th>Title</Th>
                        <Th>Artist</Th>
                        <Th>Album</Th>
                        <Th w="80px">Year</Th>
                        <Th w="90px">Flags</Th>
                    </Tr>
                </Thead>
                <Tbody>
                    {tracks.map((track) => {
                        const sel = selected[track.id];
                        return (
                            <Tr key={track.id} _hover={{ bg: 'whiteAlpha.50' }}>
                                <Td>
                                    <Checkbox
                                        isChecked={!!sel}
                                        onChange={() => toggle(track)}
                                        colorScheme="blue"
                                    />
                                </Td>
                                <Td minW="180px">
                                    {sel ? (
                                        <Input
                                            size="sm"
                                            value={sel.title}
                                            onChange={(e) => updateField(track.id, 'title', e.target.value)}
                                        />
                                    ) : (
                                        <Text>{track.title || 'Unknown'}</Text>
                                    )}
                                </Td>
                                <Td minW="160px">
                                    {sel ? (
                                        <Input
                                            size="sm"
                                            value={sel.artist}
                                            onChange={(e) => updateField(track.id, 'artist', e.target.value)}
                                        />
                                    ) : (
                                        <Text>{track.artistName || 'Unknown'}</Text>
                                    )}
                                </Td>
                                <Td minW="160px">
                                    {sel ? (
                                        <Input
                                            size="sm"
                                            value={sel.album}
                                            onChange={(e) => updateField(track.id, 'album', e.target.value)}
                                        />
                                    ) : (
                                        <Text>{track.albumName || '—'}</Text>
                                    )}
                                </Td>
                                <Td>
                                    {sel ? (
                                        <Input
                                            size="sm"
                                            type="number"
                                            value={sel.year}
                                            onChange={(e) => updateField(track.id, 'year', Number(e.target.value) || 0)}
                                        />
                                    ) : (
                                        <Text>{track.year || '—'}</Text>
                                    )}
                                </Td>
                                <Td>
                                    <HStack spacing={1}>
                                        {track.isVideo && <Badge colorScheme="orange" fontSize="xs">video</Badge>}
                                        {track.isAtmos && <Badge colorScheme="purple" fontSize="xs">atmos</Badge>}
                                        {track.isSingle && <Badge colorScheme="teal" fontSize="xs">single</Badge>}
                                        {track.explicit && <Badge colorScheme="red" fontSize="xs">E</Badge>}
                                    </HStack>
                                </Td>
                            </Tr>
                        );
                    })}
                </Tbody>
            </Table>
        </Box>
    );
};

// ---------------------------------------------------------------------------
// Import progress
// ---------------------------------------------------------------------------
const ImportProgress = ({ job }) => {
    const pct = job.total > 0 ? Math.round(((job.done + job.failed) / job.total) * 100) : 0;
    return (
        <Box
            p={5}
            border="1px solid"
            borderColor="gray.700"
            borderRadius="md"
            bg="gray.800"
        >
            <VStack spacing={3} align="stretch">
                <HStack justify="space-between">
                    <HStack spacing={2}>
                        {job.status === 'running' && <Spinner size="sm" color="blue.400" />}
                        <Text fontWeight="bold">
                            Import {job.status === 'running' ? 'in progress' : 'complete'}
                        </Text>
                    </HStack>
                    <Text fontSize="sm" color="gray.400">
                        {job.done + job.failed}/{job.total} · {job.done} ok · {job.failed} failed
                    </Text>
                </HStack>
                <Progress value={pct} size="sm" colorScheme="blue" />
                <Box maxH="240px" overflowY="auto">
                    {job.items.map((it, i) => (
                        <HStack key={i} spacing={2} py={1} fontSize="sm">
                            {it.status === 'done' && <FiCheck color="green.400" />}
                            {it.status === 'failed' && <FiX color="red.400" />}
                            {it.status === 'downloading' && <Spinner size="xs" color="blue.400" />}
                            {it.status === 'pending' && <Box w="14px" />}
                            <Text flex="1" noOfLines={1} title={it.title}>{it.title || `Track ${it.track_id}`}</Text>
                            {it.error && <Text color="red.400" fontSize="xs" noOfLines={1} title={it.error}>{it.error}</Text>}
                        </HStack>
                    ))}
                </Box>
            </VStack>
        </Box>
    );
};

// ---------------------------------------------------------------------------
// Main page
// ---------------------------------------------------------------------------
const Tidal = () => {
    const toast = useToast();
    const [status, setStatus] = useState(null);
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [showSettings, setShowSettings] = useState(false);

    const [tab, setTab] = useState('tracks');
    const [library, setLibrary] = useState({ items: [], totalNumItems: 0, offset: 0, limit: 50 });
    const [loadingLib, setLoadingLib] = useState(false);
    const [detail, setDetail] = useState(null); // {type, name, tracks}
    const [loadingDetail, setLoadingDetail] = useState(false);

    const [selected, setSelected] = useState({});
    const [playlistName, setPlaylistName] = useState('');
    const [importJob, setImportJob] = useState(null);
    const importPollRef = useRef(null);

    const connected = !!status?.connected;

    const loadStatus = useCallback(async () => {
        try {
            const res = await getStatus();
            setStatus(res.data);
        } catch (e) {
            console.error('Failed to load Tidal status', e);
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        loadStatus();
    }, [loadStatus]);

    useEffect(() => () => {
        if (importPollRef.current) clearInterval(importPollRef.current);
    }, []);

    const loadLibrary = useCallback(async (kind, offset = 0) => {
        setLoadingLib(true);
        try {
            const res = await getLibrary(kind, offset, 50);
            setLibrary(res.data);
        } catch (e) {
            toast({ title: e.message || 'Failed to load library', status: 'error', duration: 3000 });
        } finally {
            setLoadingLib(false);
        }
    }, [toast]);

    useEffect(() => {
        if (connected && !detail) {
            loadLibrary(tab);
        }
    }, [connected, tab, detail, loadLibrary]);

    const openAlbum = async (albumId, name) => {
        setLoadingDetail(true);
        setSelected({});
        try {
            const res = await getAlbum(albumId);
            setDetail({ type: 'album', name, tracks: res.data.tracks || [] });
        } catch (e) {
            toast({ title: e.message || 'Failed to load album', status: 'error', duration: 3000 });
        } finally {
            setLoadingDetail(false);
        }
    };

    const openPlaylist = async (playlistId, name) => {
        setLoadingDetail(true);
        setSelected({});
        try {
            const res = await getPlaylist(playlistId);
            setDetail({ type: 'playlist', name, tracks: res.data.tracks || [] });
        } catch (e) {
            toast({ title: e.message || 'Failed to load playlist', status: 'error', duration: 3000 });
        } finally {
            setLoadingDetail(false);
        }
    };

    const handleSaveSettings = async (cfg) => {
        setSaving(true);
        try {
            const res = await saveSettings(cfg);
            setStatus((s) => ({ ...s, config: res.data.config }));
            toast({ title: 'Settings saved', status: 'success', duration: 2000 });
        } catch (e) {
            toast({ title: e.message || 'Failed to save settings', status: 'error', duration: 3000 });
        } finally {
            setSaving(false);
        }
    };

    const handleDisconnect = async () => {
        if (!confirm('Disconnect your Tidal account?')) return;
        try {
            await disconnect();
            toast({ title: 'Disconnected', status: 'success', duration: 2000 });
            loadStatus();
        } catch (e) {
            toast({ title: e.message || 'Failed to disconnect', status: 'error', duration: 3000 });
        }
    };

    const startImportPolling = (jobId) => {
        if (importPollRef.current) clearInterval(importPollRef.current);
        importPollRef.current = setInterval(async () => {
            try {
                const res = await importStatus(jobId);
                const job = res.data;
                setImportJob(job);
                if (job.status === 'done') {
                    clearInterval(importPollRef.current);
                    importPollRef.current = null;
                    toast({
                        title: `Import complete: ${job.done} ok, ${job.failed} failed`,
                        status: job.failed > 0 ? 'warning' : 'success',
                        duration: 4000,
                    });
                    // Refresh library so imported tracks appear.
                    loadLibrary(tab);
                }
            } catch (e) {
                // ignore transient errors
            }
        }, 1500);
    };

    const handleImport = async () => {
        const tracks = Object.values(selected);
        if (tracks.length === 0) {
            toast({ title: 'Select at least one track', status: 'warning', duration: 2000 });
            return;
        }
        try {
            const res = await startImport({
                tracks,
                playlist_name: playlistName.trim(),
            });
            setImportJob({
                id: res.data.jobId,
                total: tracks.length,
                done: 0,
                failed: 0,
                status: 'running',
                items: tracks.map((t) => ({ track_id: t.track_id, title: t.title, status: 'pending' })),
            });
            startImportPolling(res.data.jobId);
            setSelected({});
        } catch (e) {
            toast({ title: e.message || 'Failed to start import', status: 'error', duration: 3000 });
        }
    };

    const selectedCount = Object.keys(selected).length;
    const activeTracks = detail ? detail.tracks : library.items;

    if (loading) {
        return (
            <Box py={20} textAlign="center">
                <Spinner size="xl" />
            </Box>
        );
    }

    return (
        <Box maxW="container.xl" mx="auto" py={8}>
            <VStack spacing={6} align="stretch">
                <HStack justify="space-between">
                    <Heading as="h1" size="xl">Tidal</Heading>
                    <HStack spacing={3}>
                        <IconButton
                            icon={<FiSettings />}
                            aria-label="Settings"
                            variant="ghost"
                            onClick={() => setShowSettings((v) => !v)}
                        />
                        {connected && (
                            <Button
                                size="sm"
                                variant="outline"
                                colorScheme="red"
                                leftIcon={<FiLogOut />}
                                onClick={handleDisconnect}
                            >
                                Disconnect
                            </Button>
                        )}
                    </HStack>
                </HStack>

                {showSettings && status?.config && (
                    <SettingsPanel
                        config={status.config}
                        onSave={handleSaveSettings}
                        saving={saving}
                    />
                )}

                {!connected ? (
                    <PairingCard onConnected={loadStatus} />
                ) : (
                    <>
                        <HStack spacing={2} color="green.400" fontSize="sm">
                            <FiCheck />
                            <Text>
                                Connected as <Text as="span" fontWeight="bold">{status.tidalUsername}</Text>
                            </Text>
                        </HStack>

                        <Tabs
                            index={['tracks', 'albums', 'playlists'].indexOf(tab)}
                            onChange={(i) => {
                                setTab(['tracks', 'albums', 'playlists'][i]);
                                setDetail(null);
                                setSelected({});
                            }}
                            colorScheme="blue"
                            isLazy={false}
                        >
                            <TabList>
                                <Tab><HStack spacing={2}><FiMusic />Tracks</HStack></Tab>
                                <Tab><HStack spacing={2}><FiDisc />Albums</HStack></Tab>
                                <Tab><HStack spacing={2}><FiList />Playlists</HStack></Tab>
                            </TabList>
                            <TabPanels>
                                <TabPanel px={0}>
                                    {loadingLib ? (
                                        <Spinner size="lg" my={10} alignSelf="center" />
                                    ) : (
                                        <TrackImportTable
                                            tracks={library.items}
                                            selected={selected}
                                            setSelected={setSelected}
                                        />
                                    )}
                                    {library.totalNumItems > library.limit && (
                                        <HStack justify="flex-end" mt={3}>
                                            <Button
                                                size="sm"
                                                variant="outline"
                                                onClick={() => loadLibrary(tab, library.offset + library.limit)}
                                            >
                                                Load more
                                            </Button>
                                        </HStack>
                                    )}
                                </TabPanel>

                                <TabPanel px={0}>
                                    {loadingLib ? (
                                        <Spinner size="lg" my={10} alignSelf="center" />
                                    ) : detail ? (
                                        <VStack spacing={4} align="stretch">
                                            <HStack>
                                                <Button size="sm" variant="ghost" onClick={() => setDetail(null)}>
                                                ← Back
                                                </Button>
                                                <Text fontWeight="bold">{detail.name}</Text>
                                            </HStack>
                                            <TrackImportTable
                                                tracks={detail.tracks}
                                                selected={selected}
                                                setSelected={setSelected}
                                            />
                                        </VStack>
                                    ) : (
                                        <Table variant="simple" size="sm">
                                            <Thead>
                                                <Tr>
                                                    <Th>Album</Th>
                                                    <Th>Artist</Th>
                                                    <Th w="80px">Year</Th>
                                                </Tr>
                                            </Thead>
                                            <Tbody>
                                                {library.items.map((a) => (
                                                    <Tr
                                                        key={a.id}
                                                        cursor="pointer"
                                                        _hover={{ bg: 'whiteAlpha.50' }}
                                                        onClick={() => openAlbum(a.id, a.title)}
                                                    >
                                                        <Td fontWeight="bold">{a.title}</Td>
                                                        <Td>{a.artistName}</Td>
                                                        <Td>{a.year || '—'}</Td>
                                                    </Tr>
                                                ))}
                                            </Tbody>
                                        </Table>
                                    )}
                                </TabPanel>

                                <TabPanel px={0}>
                                    {loadingLib ? (
                                        <Spinner size="lg" my={10} alignSelf="center" />
                                    ) : detail ? (
                                        <VStack spacing={4} align="stretch">
                                            <HStack>
                                                <Button size="sm" variant="ghost" onClick={() => setDetail(null)}>
                                                ← Back
                                                </Button>
                                                <Text fontWeight="bold">{detail.name}</Text>
                                            </HStack>
                                            <TrackImportTable
                                                tracks={detail.tracks}
                                                selected={selected}
                                                setSelected={setSelected}
                                            />
                                        </VStack>
                                    ) : (
                                        <Table variant="simple" size="sm">
                                            <Thead>
                                                <Tr>
                                                    <Th>Playlist</Th>
                                                    <Th>Artist</Th>
                                                </Tr>
                                            </Thead>
                                            <Tbody>
                                                {library.items.map((p) => (
                                                    <Tr
                                                        key={p.id}
                                                        cursor="pointer"
                                                        _hover={{ bg: 'whiteAlpha.50' }}
                                                        onClick={() => openPlaylist(p.id, p.title)}
                                                    >
                                                        <Td fontWeight="bold">{p.title}</Td>
                                                        <Td>{p.artistName}</Td>
                                                    </Tr>
                                                ))}
                                            </Tbody>
                                        </Table>
                                    )}
                                </TabPanel>
                            </TabPanels>
                        </Tabs>

                        <Box
                            p={4}
                            border="1px solid"
                            borderColor="gray.700"
                            borderRadius="md"
                            bg="gray.800"
                        >
                            <HStack spacing={4} align="flex-end" flexWrap="wrap">
                                <FormControl maxW="320px" flex="1">
                                    <FormLabel fontSize="sm" mb={1}>
                                        Add to playlist (optional)
                                    </FormLabel>
                                    <Input
                                        placeholder="My Tidal Imports"
                                        value={playlistName}
                                        onChange={(e) => setPlaylistName(e.target.value)}
                                    />
                                </FormControl>
                                <Button
                                    colorScheme="blue"
                                    leftIcon={<FiDownload />}
                                    isDisabled={selectedCount === 0}
                                    onClick={handleImport}
                                >
                                    Import {selectedCount > 0 ? `${selectedCount} ` : ''}selected
                                </Button>
                            </HStack>
                        </Box>

                        {importJob && <ImportProgress job={importJob} />}
                    </>
                )}
            </VStack>
        </Box>
    );
};

export default Tidal;
