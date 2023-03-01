import { TrackDto } from './track.dto';

export class CurrentTrackDto extends TrackDto {
  progressMs: number;
  isPlaying: boolean;
}
