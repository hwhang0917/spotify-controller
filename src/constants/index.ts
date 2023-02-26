/** Spotify Client ID Store Key */
export const SPOTIFY_CLIENT_ID = 'SPOTIFY_CLIENT_ID';
/** Spotify Secret Store Key */
export const SPOTIFY_SECRET = 'SPOTIFY_SECRET';
/** Spotify Access Token Store Key */
export const SPOTIFY_ACCESS_TOKEN = 'SPOTIFY_ACCESS_TOKEN';
/** Spotify Refresh Token Store Key */
export const SPOTIFY_REFRESH_TOKEN = 'SPOTIFY_REFRESH_TOKEN';
/** Spotify OAuth State Key */
export const SPOTIFY_STATE = 'SPOTIFY_STATE';
/** Spotify Account Base URL */
export const SPOTIFY_ACCOUNT_URL = 'https://accounts.spotify.com';
/** Spotify API Base URL */
export const SPOTIFY_API_URL = 'https://api.spotify.com';

/** DEFAULT PORT */
export const DEFAULT_PORT = 5555;
/** Redirect URL */
export const REDIRECT_URI = `http://localhost:${DEFAULT_PORT}/callback`;
/** Pagination Page Size */
export const PAGE_SIZE = 10;

/** No Album Cover Placeholder Image */
export const ALBUM_COVER_PLACEHOLDER =
  'https://via.placeholder.com/512.png/202020/ffffff?text=No%20Album%20Cover';

/** Locale Regex */
export const LOCALE_REGEX =
  /^[A-Za-z]{2,4}([_-][A-Za-z]{4})?([_-]([A-Za-z]{2}|[0-9]{3}))?$/;
