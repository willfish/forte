// Mock @wailsio/runtime for Playwright e2e tests.
// Maps Wails Call.ByID numeric identifiers to fixture responses.

let appPreferences = {
  libraryEnabled: false,
  startLastStation: true,
  autoReconnect: true,
  showTitlebar: false,
};

let radioFavourites = [
  { stationUuid: "st-1", name: "Jazz FM", streamUrl: "https://stream.example.com/jazz", faviconUrl: "", tags: "jazz,smooth", addedAt: "2024-01-01 00:00:00", pinned: false },
];

let customStations: any[] = [];

let radioHistory = [
  { stationUuid: "st-3", name: "Classical 24", streamUrl: "https://stream.example.com/classical", faviconUrl: "https://img.example.com/classical.png", tags: "classical", lastTitle: "Morning Concert", lastError: "", playCount: 3, lastPlayedAt: "2024-01-02 09:00:00" },
];

let playbackStatus = {
  state: "stopped",
  position: 0,
  duration: 0,
  volume: 80,
  title: "",
  artist: "",
  album: "",
  shuffle: false,
  repeat: "off",
  mediaPath: "",
  radioMode: false,
  radioUuid: "",
  radioStation: "",
  radioArtwork: "",
};

function stationIcon(label: string, background: string, foreground = "ffffff") {
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="96" height="96" viewBox="0 0 96 96">
    <rect width="96" height="96" rx="12" fill="#${background}"/>
    <text x="48" y="58" text-anchor="middle" font-family="Arial, sans-serif" font-size="24" font-weight="700" fill="#${foreground}">${label}</text>
  </svg>`;
  return `data:image/svg+xml,${encodeURIComponent(svg)}`;
}

const demoStations = [
  { uuid: "somafm-missioncontrol", name: "Ishq - Iqqoa", streamUrl: "https://stream.example.com/mission-control", favicon: stationIcon("MC", "1f2937"), country: "SomaFM Mission Control", tags: "ambient,space,electronic", bitrate: 128, codec: "MP3" },
  { uuid: "st-jazz-fm", name: "Jazz FM", streamUrl: "https://stream.example.com/jazz-fm", favicon: stationIcon("JF", "2563eb"), country: "United Kingdom", tags: "jazz,smooth", bitrate: 128, codec: "MP3" },
  { uuid: "st-classical-24", name: "Classical 24", streamUrl: "https://stream.example.com/classical", favicon: stationIcon("C24", "7c2d12"), country: "France", tags: "classical,orchestral", bitrate: 320, codec: "MP3" },
  { uuid: "st-mangoradio", name: "MANGORADIO", streamUrl: "https://stream.example.com/mango", favicon: stationIcon("M", "f59e0b", "111827"), country: "Germany", tags: "music,variety", bitrate: 128, codec: "MP3" },
  { uuid: "st-radio-paradise", name: "Radio Paradise Main Mix (EU) 320k AAC", streamUrl: "https://stream.example.com/rp", favicon: stationIcon("RP", "f8fafc", "334155"), country: "The United States Of America", tags: "california,eclectic,free,internet", bitrate: 320, codec: "AAC" },
  { uuid: "st-classic-vinyl", name: "Classic Vinyl HD", streamUrl: "https://stream.example.com/classic-vinyl", favicon: stationIcon("CV", "dc2626"), country: "The United States Of America", tags: "1930,1940,1950,1960", bitrate: 320, codec: "MP3" },
  { uuid: "st-jazz-underground", name: "Adroit Jazz Underground", streamUrl: "https://stream.example.com/jazz", favicon: stationIcon("JZ", "7c3aed"), country: "The United States Of America", tags: "avant-garde,bebop,big band,bop", bitrate: 320, codec: "MP3" },
  { uuid: "st-bbc-world", name: "BBC World Service", streamUrl: "https://stream.example.com/bbc", favicon: stationIcon("BBC", "ef4444"), country: "United Kingdom", tags: "news,talk,eclectic", bitrate: 56, codec: "MP3" },
  { uuid: "st-walm-old-time", name: "WALM - Old Time Radio", streamUrl: "https://stream.example.com/walm", favicon: stationIcon("OTR", "22c55e", "052e16"), country: "The United States Of America", tags: "78,78-rpm,classic", bitrate: 64, codec: "MP3" },
];

function upsertHistory(stationUuid: string, name: string, streamUrl: string, faviconUrl: string, tags: string) {
  const existing = radioHistory.find((h) => h.stationUuid === stationUuid);
  if (existing) {
    existing.playCount += 1;
    existing.lastPlayedAt = "2024-01-03 10:00:00";
    existing.lastError = "";
    return;
  }
  radioHistory.unshift({
    stationUuid,
    name,
    streamUrl,
    faviconUrl,
    tags,
    lastTitle: "",
    lastError: "",
    playCount: 1,
    lastPlayedAt: "2024-01-03 10:00:00",
  });
}

function hasTag(tags: string, tag: string) {
  if (!tag) return true;
  return tags.split(",").map((t) => t.trim().toLowerCase()).includes(tag.toLowerCase());
}

function filteredDemoStations(country = "", codec = "", tag = "") {
  return demoStations.filter((station) =>
    (!country || station.country === country) &&
    (!codec || station.codec.toLowerCase() === codec.toLowerCase()) &&
    hasTag(station.tags, tag)
  );
}

const fixtures: Record<number, (...args: any[]) => any> = {
  // --- LibraryService ---
  // GetAlbums
  1337880606: () => [
    { id: 1, title: "OK Computer", artist: "Radiohead", year: 1997, trackCount: 12, source: "local", serverId: "" },
    { id: 2, title: "Kid A", artist: "Radiohead", year: 2000, trackCount: 10, source: "local", serverId: "" },
    { id: 3, title: "Homogenic", artist: "Bjork", year: 1997, trackCount: 10, source: "server", serverId: "srv-1" },
  ],
  // AlbumArtwork
  866920135: () => "",
  // GetAlbumTracks
  207489145: () => [
    { trackId: 1, title: "Airbag", artist: "Radiohead", trackNumber: 1, discNumber: 1, durationMs: 282000, filePath: "/music/airbag.flac", source: "local", serverId: "" },
    { trackId: 2, title: "Paranoid Android", artist: "Radiohead", trackNumber: 2, discNumber: 1, durationMs: 386000, filePath: "/music/paranoid.flac", source: "local", serverId: "" },
    { trackId: 3, title: "Subterranean Homesick Alien", artist: "Radiohead", trackNumber: 3, discNumber: 1, durationMs: 267000, filePath: "/music/sha.flac", source: "local", serverId: "" },
  ],
  // Search
  2206755262: (query: string) => {
    if (!query) return [];
    return [
      { trackId: 1, title: "Airbag", artist: "Radiohead", album: "OK Computer", genre: "Rock", trackNumber: 1, discNumber: 1, durationMs: 282000, filePath: "/music/airbag.flac", source: "local", serverId: "" },
    ];
  },
  // GetPlaylists
  1524576557: () => [
    { id: 1, name: "Favourites" },
    { id: 2, name: "Chill" },
  ],
  // CreatePlaylist
  4167498172: () => 3,
  // RenamePlaylist
  3081673158: () => undefined,
  // DeletePlaylist
  2018893399: () => undefined,
  // GetPlaylistTracks
  4244880336: () => [
    { trackId: 1, title: "Airbag", artist: "Radiohead", album: "OK Computer", durationMs: 282000, filePath: "/music/airbag.flac", position: 0 },
  ],
  // AddTrackToPlaylist
  2287316659: () => undefined,
  // RemoveTrackFromPlaylist
  970681807: () => undefined,
  // MoveTrackInPlaylist
  465154155: () => undefined,
  // GetServers
  3711954650: () => [],
  // AddServer
  477958106: () => undefined,
  // UpdateServer
  1667032524: () => undefined,
  // DeleteServer
  3862467038: () => undefined,
  // GetServerStatuses
  1839345631: () => [],
  // TestConnection
  3263505778: () => undefined,
  // SyncServers
  545152779: () => undefined,
  // GetScrobbleConfig
  3948527462: () => ({ apiKey: "", sessionKey: "", username: "", enabled: false }),
  // SaveScrobbleAPIKeys
  1590775235: () => undefined,
  // StartLastFmAuth
  1558738173: () => "mock-token",
  // CompleteLastFmAuth
  3698942302: () => undefined,
  // DisconnectLastFm
  970487533: () => undefined,
  // SetScrobbleEnabled
  22544365: () => undefined,
  // GetListenBrainzConfig
  1867711289: () => ({ username: "", enabled: false }),
  // ConnectListenBrainz
  1138196949: () => undefined,
  // DisconnectListenBrainz
  902147985: () => undefined,
  // SetListenBrainzEnabled
  333272068: () => undefined,
  // GetScrobbleQueueSize
  4199289054: () => 0,
  // GetTopArtists
  2628386383: () => [
    { name: "Radiohead", secondLine: "", playCount: 42, totalMs: 5040000 },
    { name: "Bjork", secondLine: "", playCount: 18, totalMs: 2160000 },
  ],
  // GetTopAlbums
  1740480677: () => [
    { name: "OK Computer", secondLine: "Radiohead", playCount: 30, totalMs: 3600000 },
  ],
  // GetTopTracks
  3437861925: () => [
    { name: "Airbag", secondLine: "Radiohead", playCount: 15, totalMs: 4230000 },
  ],
  // GetRecentlyPlayed
  3884039413: () => [
    { trackId: 1, title: "Airbag", artist: "Radiohead", album: "OK Computer", durationMs: 282000, playedAt: new Date().toISOString().replace("T", " ").slice(0, 19) },
  ],
  // GetArtistByName
  1148779767: () => 1,
  // GetArtistInfo
  2345670893: () => ({
    name: "Radiohead", bio: "English rock band from Oxfordshire.", imageUrl: "",
    area: "Oxfordshire", type: "Group", activeYears: "1985 - present",
    similar: [{ name: "Bjork", inLibrary: true }],
    albums: [{ id: 1, title: "OK Computer", artist: "Radiohead", year: 1997, trackCount: 12, source: "local", serverId: "" }],
    tags: "alternative, rock, electronic",
  }),

  // ProxyImageURL
  1305746552: (url: string) => url || "",
  // SearchRadioStations
  1619368624: () => demoStations.map((station, index) => ({ ...station, votes: 200 - index * 10, clicks: 500 - index * 20 })),
  // GetTopVotedRadioStations
  1723581283: () => demoStations.map((station, index) => ({ ...station, votes: 200 - index * 10, clicks: 500 - index * 20 })),
  // GetTopClickedRadioStations
  46869912: () => demoStations.map((station, index) => ({ ...station, votes: 200 - index * 10, clicks: 500 - index * 20 })),
  // SearchRadioStationsFiltered
  2804279923: (country: string, codec: string, tag: string) => filteredDemoStations(country, codec, tag).map((station, index) => ({ ...station, votes: 200 - index * 10, clicks: 500 - index * 20 })),
  // GetRadioStationsByTag
  3897998615: () => [],
  // GetRadioStationsByCountry
  3988982917: () => [],
  // GetRadioFavourites
  590575721: () => radioFavourites,
  // AddRadioFavourite
  3744144887: (stationUuid: string, name: string, streamUrl: string, faviconUrl: string, tags: string) => {
    if (!radioFavourites.some((f) => f.stationUuid === stationUuid)) {
      radioFavourites.push({ stationUuid, name, streamUrl, faviconUrl, tags, addedAt: "2024-01-03 10:00:00", pinned: false });
    }
  },
  // RemoveRadioFavourite
  876184048: (stationUuid: string) => {
    radioFavourites = radioFavourites.filter((f) => f.stationUuid !== stationUuid);
  },
  // SetRadioFavouritePinned
  2949163422: (stationUuid: string, pinned: boolean) => {
    radioFavourites = radioFavourites
      .map((f) => f.stationUuid === stationUuid ? { ...f, pinned } : f)
      .sort((a, b) => Number(b.pinned) - Number(a.pinned) || a.name.localeCompare(b.name));
  },
  // IsRadioFavourite
  329793224: (stationUuid: string) => radioFavourites.some((f) => f.stationUuid === stationUuid),
  // GetCustomRadioStations
  2495430549: () => customStations,
  // AddCustomRadioStation
  3781401643: (name: string, streamUrl: string, faviconUrl: string, tags: string) => {
    const station = {
      stationUuid: `custom-${customStations.length + 1}`,
      name,
      streamUrl,
      faviconUrl,
      tags,
      createdAt: "2024-01-03 10:00:00",
    };
    customStations.push(station);
    return station;
  },
  // DeleteCustomRadioStation
  3260428599: (stationUuid: string) => {
    customStations = customStations.filter((s) => s.stationUuid !== stationUuid);
  },
  // GetRadioHistory
  3857005579: () => radioHistory,
  // ClearRadioHistory
  2733979162: () => {
    radioHistory = [];
  },
  // GetAppPreferences
  3910505449: () => appPreferences,
  // SaveAppPreferences
  1128588116: (prefs: typeof appPreferences) => {
    appPreferences = { ...appPreferences, ...prefs };
  },
  // SetThemePreference
  3763400674: () => undefined,

  // --- PlayerService ---
  // GetPlaybackStatus
  958915679: () => playbackStatus,
  // State
  2570357237: () => playbackStatus.state,
  // PlayRadio
  1236378929: () => undefined,
  // PlayRadioStation
  3331506535: (stationUuid: string, name: string, streamUrl: string, artworkUrl: string, tags: string) => {
    if (globalThis.localStorage?.getItem("forte.failPlayRadioStation") === "true") {
      throw new Error("stream unavailable");
    }
    upsertHistory(stationUuid, name, streamUrl, artworkUrl, tags);
    playbackStatus = {
      ...playbackStatus,
      state: "playing",
      title: name,
      artist: stationUuid === "somafm-missioncontrol" ? "SomaFM Mission Control (128k MP3)" : "Radio",
      album: "",
      mediaPath: streamUrl,
      radioMode: true,
      radioUuid: stationUuid,
      radioStation: stationUuid === "somafm-missioncontrol" ? "SomaFM Mission Control (128k MP3)" : name,
      radioArtwork: artworkUrl,
    };
  },
  // StopRadio
  3776601259: () => undefined,
  // IsRadioMode
  2964685828: () => false,
  // RadioStationName
  2576550642: () => "",
  // RadioArtworkURL
  1282363134: () => "",
  // Play
  1808111650: () => undefined,
  // Pause
  191671602: () => {
    if (playbackStatus.state === "playing") {
      playbackStatus = { ...playbackStatus, state: "paused" };
    }
  },
  // Resume
  4192344979: () => {
    if (playbackStatus.state === "paused") {
      playbackStatus = { ...playbackStatus, state: "playing" };
    }
  },
  // Stop
  2311398648: () => {
    playbackStatus = { ...playbackStatus, state: "stopped", title: "", artist: "", album: "", mediaPath: "", radioMode: false, radioUuid: "" };
  },
  // Seek
  1479346536: () => undefined,
  // Position
  3379668963: () => 0,
  // Duration
  1985222848: () => 0,
  // Volume
  2798880050: () => 80,
  // SetVolume
  671101282: () => undefined,
  // GetShuffle
  4278779269: () => false,
  // SetShuffle
  3896707945: () => undefined,
  // GetRepeat
  3949558547: () => "off",
  // SetRepeat
  76083775: () => undefined,
  // Next
  1009561457: () => undefined,
  // Previous
  2487521925: () => undefined,
  // MediaTitle
  3116228434: () => playbackStatus.title,
  // MediaArtist
  3929664599: () => playbackStatus.artist,
  // MediaAlbum
  3994078579: () => playbackStatus.album,
  // MediaPath
  3316771859: () => playbackStatus.mediaPath,
  // Artwork
  468839008: () => "",
  // Enqueue
  1683307842: () => undefined,
  // PlayAll
  3674799417: () => undefined,
  // PlayQueue
  3857677157: (tracks: any[], startAt: number) => {
    const track = tracks[startAt] || tracks[0];
    if (track) {
      playbackStatus = {
        ...playbackStatus,
        state: "playing",
        position: 0,
        duration: Math.floor((track.durationMs || 0) / 1000),
        title: track.title || "",
        artist: track.artist || "",
        album: track.album || "",
        mediaPath: track.filePath || "",
        radioMode: false,
        radioStation: "",
        radioArtwork: "",
      };
    }
  },
  // GetQueue
  1525514291: () => [],
  // GetQueuePosition
  752141504: () => -1,
  // QueueAppend
  3799532135: () => undefined,
  // QueueInsertNext
  125730107: () => undefined,
  // QueueRemove
  1743380467: () => undefined,
  // QueueMove
  3873105318: () => undefined,
  // QueueClear
  3052298016: () => undefined,
  // GetNotifications
  3578942832: () => false,
  // SetNotifications
  2522355060: () => undefined,
  // GetToasts
  327853480: () => [],
  // ReplayGain
  3252990072: () => "no",
  // SetReplayGain
  804885384: () => undefined,
  // Version
  1040332204: () => "mock-1.0",
};

class MockCancellablePromise<T> extends Promise<T> {
  cancel() {}
}

export const Call = {
  ByID(id: number, ...args: any[]): MockCancellablePromise<any> {
    const handler = fixtures[id];
    if (handler) {
      return MockCancellablePromise.resolve(handler(...args));
    }
    console.warn(`[wails-mock] Unhandled Call.ByID: ${id}`);
    return MockCancellablePromise.resolve(null);
  },
};

export const CancellablePromise = MockCancellablePromise;

export const Window = {
  Minimise: () => MockCancellablePromise.resolve(undefined),
  ToggleMaximise: () => MockCancellablePromise.resolve(undefined),
  Close: () => MockCancellablePromise.resolve(undefined),
};

export const Create = {
  Array(createFn: (source: any) => any) {
    return (arr: any[]) => (arr ?? []).map(createFn);
  },
};

export default { Call, CancellablePromise, Create, Window };
