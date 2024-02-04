/**
 * Default configuration for the SpotifyController
 */
export const DEFAULT_CONFIG: SpotifyControllerConfig = {
  allowViewing: true,
  allowPausing: false,
  allowSkipping: false,
  language: "english",
};

/** Spotify Account Base URL */
export const SPOTIFY_ACCOUNT_URL = "https://accounts.spotify.com";
/** Spotify API Base URL */
export const SPOTIFY_API_URL = "https://api.spotify.com";

/** Spotify API Scope */
export const SPOTIFY_SCOPE =
  "user-read-playback-state user-modify-playback-state user-read-currently-playing";
/** Spotiufy Redirect URL */
export const SPOTIFY_REDIRECT_ENDPOINT = "/callback";

/** Pagination Page Size */
export const PAGE_SIZE = 20;

/** No Album Cover Placeholder Image */
export const ALBUM_COVER_PLACEHOLDER =
  "https://via.placeholder.com/512.png/202020/ffffff?text=No%20Album%20Cover";

/** Default Locale */
export const DEFAULT_LOCALE = "en_US";

/* Spotify Access Token Key */
export const SPOTIFY_ACCESS_TOKEN = "SPOTIFY_ACCESS_TOKEN";
/* Spotify Refresh Token Key */
export const SPOTIFY_REFRESH_TOKEN = "SPOTIFY_REFRESH_TOKEN";
