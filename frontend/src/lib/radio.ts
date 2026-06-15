import type { PlaybackState } from "./types";

export type Station = {
  uuid: string;
  name: string;
  streamUrl: string;
  homepage: string;
  favicon: string;
  country: string;
  tags: string;
  bitrate: number;
  codec: string;
  votes: number;
  clicks: number;
};

export type Favourite = {
  stationUuid: string;
  name: string;
  streamUrl: string;
  faviconUrl: string;
  homepage: string;
  country: string;
  codec: string;
  bitrate: number;
  tags: string;
  addedAt: string;
  pinned: boolean;
};

export type CustomStation = {
  stationUuid: string;
  name: string;
  streamUrl: string;
  faviconUrl: string;
  homepage: string;
  tags: string;
  createdAt: string;
};

export type HistoryEntry = {
  stationUuid: string;
  name: string;
  streamUrl: string;
  faviconUrl: string;
  homepage: string;
  country: string;
  codec: string;
  bitrate: number;
  tags: string;
  lastTitle: string;
  lastError: string;
  playCount: number;
  lastPlayedAt: string;
};

export type RadioCountry = {
  code: string;
  label: string;
  names: string[];
};

export type RadioFilters = {
  activeTags: string[];
  activeSource: "all" | "somafm";
  activeCountry: string;
  activeCodec: string;
};

export type PlayStationInput = {
  stationUuid: string;
  name: string;
  streamUrl: string;
  favicon: string;
  homepage: string;
  tags: string;
  country?: string;
  codec?: string;
  bitrate?: number;
};

export type PlaybackSnapshot = {
  radioMode: boolean;
  currentStationUuid: string;
  playbackState: PlaybackState;
};

type RadioLibraryService = {
  AddCustomRadioStation(name: string, streamURL: string, faviconURL: string, homepage: string, tags: string): Promise<any>;
  AddRadioFavourite(stationUUID: string, name: string, streamURL: string, faviconURL: string, homepage: string, tags: string, country: string, codec: string, bitrate: number): Promise<void>;
  ClearRadioHistory(): Promise<void>;
  DeleteCustomRadioStation(stationUUID: string): Promise<void>;
  GetCustomRadioStations(): Promise<any[]>;
  GetRadioFavourites(): Promise<any[]>;
  GetRadioHistory(limit: number): Promise<any[]>;
  GetSomaFMStations(): Promise<any[]>;
  ProxyImageURL(url: string): Promise<string>;
  RemoveRadioFavourite(stationUUID: string): Promise<void>;
  SearchRadioStations(query: string, limit: number): Promise<any[]>;
  SearchRadioStationsFiltered(country: string, codec: string, tag: string, limit: number): Promise<any[]>;
  SetRadioFavouritePinned(stationUUID: string, pinned: boolean): Promise<void>;
};

type RadioPlayerService = {
  Pause(): Promise<void>;
  PlayRadioStation(stationUUID: string, name: string, streamURL: string, artworkURL: string, homepage: string, tags: string, country: string, codec: string, bitrate: number): Promise<void>;
  Resume(): Promise<void>;
};

export type RadioServices = {
  library: Partial<RadioLibraryService>;
  player: Partial<RadioPlayerService>;
  refreshPlaybackStatus: () => Promise<void>;
};

export const radioCountries: RadioCountry[] = [
  { code: "US", label: "US", names: ["The United States Of America", "United States"] },
  { code: "GB", label: "UK", names: ["United Kingdom", "The United Kingdom Of Great Britain And Northern Ireland"] },
  { code: "DE", label: "DE", names: ["Germany"] },
  { code: "FR", label: "FR", names: ["France"] },
  { code: "CA", label: "CA", names: ["Canada"] },
  { code: "AU", label: "AU", names: ["Australia"] },
];

export const radioCodecs = ["MP3", "AAC", "OGG"];

let defaultServicesPromise: Promise<RadioServices> | null = null;

export function createRadioClient(services?: RadioServices) {
  const useServices = () => services ? Promise.resolve(services) : defaultServices();

  const getHistory = async (limit = 50): Promise<HistoryEntry[]> => {
    const currentServices = await useServices();
    const result = await currentServices.library.GetRadioHistory!(limit);
    return (result || []).map(mapHistoryEntry);
  };

  const getFavourites = async (): Promise<Favourite[]> => {
    const currentServices = await useServices();
    const result = await currentServices.library.GetRadioFavourites!();
    return (result || []).map(mapFavourite);
  };

  return {
    async proxyImageURL(url: string): Promise<string> {
      const currentServices = await useServices();
      return currentServices.library.ProxyImageURL!(url);
    },

    async getFeaturedStations(countries: RadioCountry[] = radioCountries): Promise<Station[]> {
      const currentServices = await useServices();
      const perCountry = await Promise.all(
        countries.map((country) =>
          currentServices.library.SearchRadioStationsFiltered!(country.code, "", "", 20)
            .then((stations) => (stations || []).map(mapStation))
            .catch(() => [] as Station[]),
        ),
      );
      const seen = new Set<string>();
      const merged: Station[] = [];
      for (const batch of perCountry) {
        for (const station of batch) {
          if (!seen.has(station.uuid)) {
            seen.add(station.uuid);
            merged.push(station);
          }
        }
      }
      merged.sort((a, b) => b.votes - a.votes);
      return merged;
    },

    async getSomaFMStations(filters: RadioFilters): Promise<Station[]> {
      const currentServices = await useServices();
      const result = await currentServices.library.GetSomaFMStations!();
      return (result || []).map(mapStation).filter((station) => stationMatchesActiveFilters(station, filters));
    },

    async getFilteredStations(filters: RadioFilters): Promise<Station[]> {
      if (filters.activeSource === "somafm") {
        return this.getSomaFMStations(filters);
      }
      const currentServices = await useServices();
      const result = await currentServices.library.SearchRadioStationsFiltered!(
        filters.activeCountry,
        filters.activeCodec,
        filters.activeTags[0] || "",
        100,
      );
      return (result || []).map(mapStation).filter((station) => stationMatchesActiveFilters(station, filters));
    },

    async searchStations(query: string): Promise<Station[]> {
      const currentServices = await useServices();
      const result = await currentServices.library.SearchRadioStations!(query, 100);
      return (result || []).map(mapStation);
    },

    getFavourites,

    async getCustomStations(): Promise<CustomStation[]> {
      const currentServices = await useServices();
      const result = await currentServices.library.GetCustomRadioStations!();
      return (result || []).map(mapCustomStation);
    },

    getHistory,

    async playStation(station: PlayStationInput, playback: PlaybackSnapshot): Promise<HistoryEntry[] | null> {
      const currentServices = await useServices();
      if (isCurrentStation(station.stationUuid, playback)) {
        if (playback.playbackState === "playing") {
          await currentServices.player.Pause!();
        } else if (playback.playbackState === "paused") {
          await currentServices.player.Resume!();
        }
        await currentServices.refreshPlaybackStatus();
        return null;
      }

      const art = station.favicon ? await currentServices.library.ProxyImageURL!(station.favicon) : "";
      await currentServices.player.PlayRadioStation!(
        station.stationUuid,
        station.name,
        station.streamUrl,
        art,
        station.homepage,
        station.tags,
        station.country || "",
        station.codec || "",
        station.bitrate || 0,
      );
      await currentServices.refreshPlaybackStatus();
      return getHistory();
    },

    async addFavourite(station: Station): Promise<Favourite[]> {
      const currentServices = await useServices();
      await currentServices.library.AddRadioFavourite!(
        station.uuid,
        station.name,
        station.streamUrl,
        station.favicon,
        station.homepage,
        station.tags,
        station.country,
        station.codec,
        station.bitrate,
      );
      return getFavourites();
    },

    async removeFavourite(uuid: string): Promise<void> {
      const currentServices = await useServices();
      await currentServices.library.RemoveRadioFavourite!(uuid);
    },

    async setFavouritePinned(favourite: Favourite): Promise<Favourite[]> {
      const currentServices = await useServices();
      await currentServices.library.SetRadioFavouritePinned!(favourite.stationUuid, !favourite.pinned);
      return getFavourites();
    },

    async saveCustomStation(name: string, streamUrl: string, faviconUrl: string, homepage: string, tags: string): Promise<CustomStation> {
      const currentServices = await useServices();
      const saved = await currentServices.library.AddCustomRadioStation!(name, streamUrl, faviconUrl, homepage, tags);
      return mapCustomStation(saved);
    },

    async deleteCustomStation(uuid: string): Promise<void> {
      const currentServices = await useServices();
      await currentServices.library.DeleteCustomRadioStation!(uuid);
    },

    async clearHistory(): Promise<void> {
      const currentServices = await useServices();
      await currentServices.library.ClearRadioHistory!();
    },
  };
}

async function defaultServices(): Promise<RadioServices> {
  defaultServicesPromise ??= Promise.all([
    import("../../bindings/github.com/willfish/forte"),
    import("./playback"),
  ]).then(([bindings, playback]) => ({
    library: bindings.LibraryService,
    player: bindings.PlayerService,
    refreshPlaybackStatus: playback.refreshPlaybackStatus,
  }));
  return defaultServicesPromise;
}

export function mapStation(station: any): Station {
  return {
    uuid: station.uuid,
    name: station.name,
    streamUrl: station.streamUrl,
    homepage: station.homepage || "",
    favicon: station.favicon,
    country: station.country,
    tags: station.tags,
    bitrate: station.bitrate,
    codec: station.codec,
    votes: station.votes,
    clicks: station.clicks,
  };
}

export function mapFavourite(favourite: any): Favourite {
  return {
    stationUuid: favourite.stationUuid,
    name: favourite.name,
    streamUrl: favourite.streamUrl,
    faviconUrl: favourite.faviconUrl,
    homepage: favourite.homepage || "",
    country: favourite.country || "",
    codec: favourite.codec || "",
    bitrate: favourite.bitrate || 0,
    tags: favourite.tags,
    addedAt: favourite.addedAt,
    pinned: Boolean(favourite.pinned),
  };
}

export function mapCustomStation(station: any): CustomStation {
  return {
    stationUuid: station.stationUuid,
    name: station.name,
    streamUrl: station.streamUrl,
    faviconUrl: station.faviconUrl,
    homepage: station.homepage || "",
    tags: station.tags,
    createdAt: station.createdAt,
  };
}

export function mapHistoryEntry(entry: any): HistoryEntry {
  return {
    stationUuid: entry.stationUuid,
    name: entry.name,
    streamUrl: entry.streamUrl,
    faviconUrl: entry.faviconUrl,
    homepage: entry.homepage || "",
    country: entry.country || "",
    codec: entry.codec || "",
    bitrate: entry.bitrate || 0,
    tags: entry.tags,
    lastTitle: entry.trackTitle || entry.lastTitle || "",
    lastError: entry.lastError,
    playCount: entry.playCount,
    lastPlayedAt: entry.lastPlayedAt,
  };
}

export function formatTags(tags: string): string[] {
  if (!tags) return [];
  return tags.split(",").map((tag) => tag.trim()).filter(Boolean).slice(0, 4);
}

function isCurrentStation(stationUuid: string, playback: PlaybackSnapshot): boolean {
  return playback.radioMode && stationUuid !== "" && playback.currentStationUuid === stationUuid;
}

function stationHasTag(station: Station, tag: string): boolean {
  if (!tag) return true;
  return formatTags(station.tags).some((stationTag) => stationTag.toLowerCase() === tag.toLowerCase());
}

function stationMatchesCountry(station: Station, countryCode: string): boolean {
  if (!countryCode) return true;
  const country = radioCountries.find((candidate) => candidate.code === countryCode);
  return country ? country.names.includes(station.country) : station.country === countryCode;
}

function stationMatchesActiveFilters(station: Station, filters: RadioFilters): boolean {
  return stationMatchesCountry(station, filters.activeCountry) &&
    (!filters.activeCodec || station.codec.toLowerCase() === filters.activeCodec.toLowerCase()) &&
    filters.activeTags.every((tag) => stationHasTag(station, tag));
}
