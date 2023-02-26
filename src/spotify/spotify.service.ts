import {
  Inject,
  Injectable,
  UnauthorizedException,
  CACHE_MANAGER,
  Logger,
} from '@nestjs/common';
import { api } from '@utils/api';
import {
  ALBUM_COVER_PLACEHOLDER,
  PAGE_SIZE,
  SPOTIFY_ACCESS_TOKEN,
} from '@constants';
import type { Cache } from 'cache-manager';
import type { ISpotifySongSearchResponse } from 'src/types/spotify-api';
import type { CurrentTrackDto, SearchTrackResultDto, TrackDto } from './dtos';
import { ISpotifyCurrentlyPlayingTrack } from 'src/types/spotify-currently-playing-trxk';
import { listFormatter } from '@utils/string';

@Injectable()
export class SpotifyService {
  private readonly logger = new Logger(SpotifyService.name);
  constructor(@Inject(CACHE_MANAGER) private readonly cacheManager: Cache) {}

  /**
   * Search Spotify songs
   */
  async searchSongs(q: string, page = 1): Promise<SearchTrackResultDto> {
    try {
      // Calculate Offset
      const offset = PAGE_SIZE * (page - 1);

      // Get Access Token
      const accessToken = await this.cacheManager.get(SPOTIFY_ACCESS_TOKEN);
      if (!accessToken) {
        throw new UnauthorizedException('login_required');
      }

      // Request Spotify API Search
      const { data } = await api<ISpotifySongSearchResponse>('/v1/search', {
        params: {
          q,
          type: 'track',
          market: 'KR',
          limit: PAGE_SIZE,
          offset,
        },
        headers: {
          Authorization: `Bearer ${accessToken}`,
        },
      });

      // Parse Data
      const parsedSongs: Array<TrackDto> = data.tracks.items.map((song) => {
        const artistsNameArray = song.artists.map(({ name }) => name);
        const artists = listFormatter.format(artistsNameArray);
        const coverImageUrl =
          song.album.images?.at(0)?.url ?? ALBUM_COVER_PLACEHOLDER;

        return {
          id: song.id,
          name: song.name,
          artists,
          previewUrl: song.preview_url,
          coverImageUrl,
          durationMs: song.duration_ms,
          explicit: song.explicit,
          href: song.href,
        };
      });
      return {
        data: parsedSongs,
        meta: {
          page,
          pageSize: PAGE_SIZE,
          total: data.tracks.total,
        },
      };
    } catch (err) {
      this.logger.error(err);
      throw err;
    }
  }

  /**
   * Get currently playing track in queue
   */
  async getCurrentlyPlayingTrack(): Promise<CurrentTrackDto> {
    try {
      // Get Access Token
      const accessToken = await this.cacheManager.get(SPOTIFY_ACCESS_TOKEN);
      if (!accessToken) {
        throw new UnauthorizedException('login_required');
      }

      // Request Spotify currently playing track API
      const { data } = await api.get<ISpotifyCurrentlyPlayingTrack>(
        '/v1/me/player/currently-playing',
        {
          params: { market: 'KR' },
          headers: {
            Authorization: `Bearer ${accessToken}`,
          },
        },
      );

      // If nothing is playing return undefined
      if (!data) {
        return undefined;
      }

      // Parse data
      const artistsNameArray = data.item.artists.map(({ name }) => name);
      const artists = listFormatter.format(artistsNameArray);
      const coverImageUrl =
        data.item.album.images?.at(0)?.url ?? ALBUM_COVER_PLACEHOLDER;

      const currentTrack: CurrentTrackDto = {
        id: data.item.id,
        artists,
        name: data.item.name,
        coverImageUrl,
        durationMs: data.item.duration_ms,
        progressMs: data.progress_ms,
        explicit: data.item.explicit,
        href: data.item.href,
        previewUrl: data.item.preview_url,
        isPlaying: data.is_playing,
      };

      return currentTrack;
    } catch (err: any) {
      console.dir(err?.response?.data);
      this.logger.error(err);
      throw err;
    }
  }

  async getQueue() {
    try {
    } catch (err) {
      this.logger.error(err);
      throw err;
    }
  }
}
