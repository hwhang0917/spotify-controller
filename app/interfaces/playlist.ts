import { ITrack } from "./track";

export interface IPlaylist {
  currentTrack: ITrack;
  queue: ITrack[];
}
