import stationsJson from "@radio-fixtures";

export type RadioStationFixture = {
  uuid: string;
  name: string;
  streamUrl: string;
  homepage: string;
  favicon: string;
  country: string;
  tags: string;
  bitrate: number;
  codec: string;
  votes?: number;
  clicks?: number;
};

export type CustomStationFixture = {
  name: string;
  streamUrl: string;
  homepage: string;
  derivedHomepage: string;
  faviconUrl: string;
  tags: string;
  country: string;
  codec: string;
  bitrate: number;
};

export type RadioFixtures = {
  radiobrowser: RadioStationFixture;
  somafm: RadioStationFixture;
  custom: CustomStationFixture;
  favouriteOnly: RadioStationFixture;
  historyOnly: RadioStationFixture;
  browse: RadioStationFixture;
};

export const radioFixtures = stationsJson as RadioFixtures;

/** Stable UUID for the custom fixture stream (matches Go CustomRadioStationUUID). */
export const customFixtureUUID = "custom-d8c4e31cbf14b18f";