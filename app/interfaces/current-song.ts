import { ITrack } from "./track";

export interface ICurrentSongResponse extends ITrack {
  /**
   * Progress in milliseconds
   */
  progressMs: number;
  /**
   * If the song is playing
   */
  isPlaying: boolean;
}
