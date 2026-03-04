// Mock @wailsio/runtime for Playwright e2e tests.
// Maps Wails Call.ByID numeric identifiers to fixture responses.

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

  // --- PlayerService ---
  // State
  2570357237: () => "stopped",
  // Play
  1808111650: () => undefined,
  // Pause
  191671602: () => undefined,
  // Resume
  4192344979: () => undefined,
  // Stop
  2311398648: () => undefined,
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
  3116228434: () => "",
  // MediaArtist
  3929664599: () => "",
  // MediaAlbum
  3994078579: () => "",
  // MediaPath
  3316771859: () => "",
  // Artwork
  468839008: () => "",
  // Enqueue
  1683307842: () => undefined,
  // PlayAll
  3674799417: () => undefined,
  // PlayQueue
  3857677157: () => undefined,
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

export const Create = {
  Array(createFn: (source: any) => any) {
    return (arr: any[]) => (arr ?? []).map(createFn);
  },
};

export default { Call, CancellablePromise, Create };
