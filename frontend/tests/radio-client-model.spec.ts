import { expect, test } from "@playwright/test";

import { createRadioClient, type RadioServices } from "../src/lib/radio";

function createServices(): RadioServices & { calls: string[] } {
  const calls: string[] = [];

  return {
    calls,
    library: {
      ProxyImageURL: async (url: string) => {
        calls.push(`proxy:${url}`);
        return `data:${url}`;
      },
      GetRadioHistory: async (_limit: number) => {
        calls.push("history");
        return [];
      },
    },
    player: {
      Pause: async () => {
        calls.push("pause");
      },
      Resume: async () => {
        calls.push("resume");
      },
      PlayRadioStation: async (...args: unknown[]) => {
        calls.push(`play:${args.join("|")}`);
      },
    },
    refreshPlaybackStatus: async () => {
      calls.push("refresh");
    },
  };
}

test("radio client resumes the current paused station without restarting the stream", async () => {
  const services = createServices();
  const client = createRadioClient(services);

  await client.playStation(
    {
      stationUuid: "station-1",
      name: "Station One",
      streamUrl: "https://stream.example/one",
      favicon: "https://img.example/one.png",
      homepage: "https://station.example",
      tags: "jazz",
    },
    {
      radioMode: true,
      currentStationUuid: "station-1",
      playbackState: "paused",
    },
  );

  expect(services.calls).toEqual(["resume", "refresh"]);
});

test("radio client starts a different station with proxied artwork and reloads history", async () => {
  const services = createServices();
  const client = createRadioClient(services);

  await client.playStation(
    {
      stationUuid: "station-2",
      name: "Station Two",
      streamUrl: "https://stream.example/two",
      favicon: "https://img.example/two.png",
      homepage: "https://station-two.example",
      tags: "soul,funk",
      country: "US",
      codec: "MP3",
      bitrate: 128,
    },
    {
      radioMode: true,
      currentStationUuid: "station-1",
      playbackState: "playing",
    },
  );

  expect(services.calls).toEqual([
    "proxy:https://img.example/two.png",
    "play:station-2|Station Two|https://stream.example/two|data:https://img.example/two.png|https://station-two.example|soul,funk|US|MP3|128",
    "refresh",
    "history",
  ]);
});
