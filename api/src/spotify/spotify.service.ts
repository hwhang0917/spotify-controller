import {
  Inject,
  Injectable,
  UnauthorizedException,
  Logger,
  CACHE_MANAGER,
  InternalServerErrorException,
  BadRequestException,
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
import { getLanguageLocale } from '@utils/locale';
import { QueueDto } from './dtos/queue.dto';
import { ISpotifyQueueResponse } from 'src/types/spotify-queue';
import { AxiosError } from 'axios';
import { CommonResponseDto } from 'src/common/dto/common-response.dto';

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
          market: getLanguageLocale().split('_').at(-1),
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
          uri: song.uri,
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
          params: { market: getLanguageLocale().split('_').at(-1) },
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
      const artistsNameArray = data.item?.artists?.map(({ name }) => name) ?? [
        'Unknown',
      ];
      const artists = listFormatter.format(artistsNameArray);
      const coverImageUrl =
        data.item?.album.images?.at(0)?.url ?? ALBUM_COVER_PLACEHOLDER;

      const currentTrack: CurrentTrackDto = {
        uri: data.item.uri,
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
      this.logger.error(err);
      throw err;
    }
  }

  async getQueue() {
    try {
      // Get Access Token
      const accessToken = await this.cacheManager.get(SPOTIFY_ACCESS_TOKEN);
      if (!accessToken) {
        throw new UnauthorizedException('login_required');
      }

      // Request Spotify Player Queue API
      const { data } = await api.get<ISpotifyQueueResponse>(
        '/v1/me/player/queue',
        {
          params: { market: getLanguageLocale().split('_').at(-1) },
          headers: {
            Authorization: `Bearer ${accessToken}`,
          },
        },
      );

      // If nothing is playing return undefined
      if (!data) {
        return undefined;
      }

      // Parse Data
      // Parse Current Track
      const artistsNameArray = data.currently_playing?.artists?.map(
        ({ name }) => name,
      ) ?? ['Unknown'];
      const artists = listFormatter.format(artistsNameArray);
      const coverImageUrl =
        data.currently_playing.album.images?.at(0)?.url ??
        ALBUM_COVER_PLACEHOLDER;
      const currentTrack: TrackDto = {
        uri: data.currently_playing.uri,
        id: data.currently_playing.id,
        artists,
        name: data.currently_playing.name,
        coverImageUrl,
        durationMs: data.currently_playing.duration_ms,
        explicit: data.currently_playing.explicit,
        href: data.currently_playing.href,
        previewUrl: data.currently_playing.preview_url,
      };
      // Parse Queue
      const queue: Array<TrackDto> = data.queue.map((song) => {
        const artistsNameArray = song.artists.map(({ name }) => name);
        const artists = listFormatter.format(artistsNameArray);
        const coverImageUrl =
          song.album.images?.at(0)?.url ?? ALBUM_COVER_PLACEHOLDER;

        return {
          uri: song.uri,
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
      const pasredData: QueueDto = {
        currentTrack,
        queue,
      };

      return pasredData;
    } catch (err: any) {
      this.logger.error(err);
      throw err;
    }
  }

  /**
   * Calls Spotify Add Item to Playback Queue API
   * @param trakcId
   */
  async addTrackToQueue(trackId: string): Promise<CommonResponseDto> {
    try {
      // Get Access Token
      const accessToken = await this.cacheManager.get(SPOTIFY_ACCESS_TOKEN);
      if (!accessToken) {
        throw new UnauthorizedException('login_required');
      }

      // Request Spotify Add Playback Queue API
      const { status } = await api({
        method: 'post',
        url: '/v1/me/player/queue',
        params: {
          uri: `spotify:track:${trackId}`,
        },
        headers: {
          Authorization: `Bearer ${accessToken}`,
        },
      });

      if (status !== 204) {
        throw new InternalServerErrorException();
      }

      return {
        success: true,
        message: `Added 'spotify:track:${trackId}' to player queue.`,
      };
    } catch (err) {
      if (err instanceof AxiosError) {
        if (400 <= err.response.status && err.response.status < 500) {
          throw new BadRequestException(err.response.data?.error?.message);
        }
      }
      this.logger.error(err);
      throw err;
    }
  }
}
