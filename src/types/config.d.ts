/**
 * Basic Configuration for the Spotify Controller
 */
interface SpotifyControllerConfig {
  /**
   * Allow the user to view the playlist
   */
  allowViewing: boolean;
  /**
   * Allow the user to pause the playlist
   */
  allowPausing: boolean;
  /**
   * Allow the user to skip the playlist
   */
  allowSkipping: boolean;
  /**
   * Language
   */
  language: Language;
}
