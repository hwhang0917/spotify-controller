import { PaginationResponseDto } from 'src/common/dto/pagination-response.dto';

export class TrackDto {
  /**
   * Spotify Unique ID
   */
  id: string;
  /**
   * Name of the track
   */
  name: string;
  /**
   * Name of the artists
   */
  artists: string;
  /**
   * Song preview URL
   */
  previewUrl: string;
  /**
   * Song cover image URL
   */
  coverImageUrl: string;
  /**
   * Duration in milliseconds
   */
  durationMs: number;
  /**
   * Explicit tag
   */
  explicit: boolean;
  /**
   * Link of the song
   */
  href: string;
}

export class SearchTrackResultDto extends PaginationResponseDto<TrackDto> {}
