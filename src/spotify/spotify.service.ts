import {
  Inject,
  Injectable,
  UnauthorizedException,
  CACHE_MANAGER,
} from '@nestjs/common';
import { api } from '@utils/api';
import {
  ALBUM_COVER_PLACEHOLDER,
  PAGE_SIZE,
  SPOTIFY_ACCESS_TOKEN,
} from '@constants';
import type { Cache } from 'cache-manager';
import type { ISpotifySongSearchResponse } from 'src/types/spotify-api';
import { SearchTrackResultDto, TrackDto } from './dtos/search-result.dto';

@Injectable()
export class SpotifyService {
  constructor(@Inject(CACHE_MANAGER) private readonly cacheManager: Cache) {}

  /**
   * Calls Spotify Search API
   */
  async searchSongs(q: string, page = 1): Promise<SearchTrackResultDto> {
    try {
      const offset = PAGE_SIZE * (page - 1);
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
        const artists = new Intl.ListFormat('ko-KR').format(artistsNameArray);
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
      throw err;
    }
  }
}
