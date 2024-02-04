interface ExternalUrls {
  spotify: string;
}

interface Artist {
  external_urls: ExternalUrls;
  href: string;
  id: string;
  name: string;
  type: string;
  uri: string;
}

interface ExternalUrls2 {
  spotify: string;
}

interface Image {
  height: number;
  url: string;
  width: number;
}

interface Album {
  album_type: string;
  artists: Artist[];
  external_urls: ExternalUrls2;
  href: string;
  id: string;
  images: Image[];
  name: string;
  release_date: string;
  release_date_precision: string;
  total_tracks: number;
  type: string;
  uri: string;
}

interface ExternalUrls3 {
  spotify: string;
}

interface Artist2 {
  external_urls: ExternalUrls3;
  href: string;
  id: string;
  name: string;
  type: string;
  uri: string;
}

interface ExternalIds {
  isrc: string;
}

interface ExternalUrls4 {
  spotify: string;
}

interface Item {
  album: Album;
  artists: Artist2[];
  disc_number: number;
  duration_ms: number;
  explicit: boolean;
  external_ids: ExternalIds;
  external_urls: ExternalUrls4;
  href: string;
  id: string;
  is_local: boolean;
  is_playable: boolean;
  name: string;
  popularity: number;
  preview_url: string;
  track_number: number;
  type: string;
  uri: string;
}

interface Tracks {
  href: string;
  items: Item[];
  limit: number;
  next: string;
  offset: number;
  previous?: any;
  total: number;
}

interface SpotifySongSearchResponse {
  tracks: Tracks;
}

interface SpotifyAccessTokenResponse {
  access_token: string;
  token_type: "Bearer";
  scope: string;
  expires_in: number;
  refresh_token: string;
}

interface SpotifyRenewAccessTokenResponse {
  access_token: string;
  token_type: "Bearer";
  scope: string;
  expires_in: number;
}

interface SpotifyCurrentlyPlayingTrack {
  timestamp: number;
  context: Context;
  progress_ms: number;
  item: Item;
  currently_playing_type: string;
  actions: Actions;
  is_playing: boolean;
}

interface Queue {
  album: Album2;
  artists: Artist4[];
  available_markets: string[];
  disc_number: number;
  duration_ms: number;
  explicit: boolean;
  external_ids: ExternalIds4;
  external_urls: ExternalUrls8;
  href: string;
  id: string;
  is_playable: boolean;
  linked_from: LinkedFrom2;
  restrictions: Restrictions4;
  name: string;
  popularity: number;
  preview_url: string;
  track_number: number;
  type: string;
  uri: string;
  is_local: boolean;
}

interface CurrentlyPlaying {
  album: Album;
  artists: Artist2[];
  available_markets: string[];
  disc_number: number;
  duration_ms: number;
  explicit: boolean;
  external_ids: ExternalIds2;
  external_urls: ExternalUrls4;
  href: string;
  id: string;
  is_playable: boolean;
  linked_from: LinkedFrom;
  restrictions: Restrictions2;
  name: string;
  popularity: number;
  preview_url: string;
  track_number: number;
  type: string;
  uri: string;
  is_local: boolean;
}

interface SpotifyQueueResponse {
  currently_playing: CurrentlyPlaying;
  queue: Queue[];
}
