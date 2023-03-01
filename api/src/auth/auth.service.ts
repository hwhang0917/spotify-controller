import qs from 'qs';
import ms from 'ms';
import axios from 'axios';
import { CACHE_MANAGER, Inject, Injectable, Logger } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import {
  SPOTIFY_ACCESS_TOKEN,
  SPOTIFY_ACCOUNT_URL,
  SPOTIFY_CLIENT_ID,
  SPOTIFY_REFRESH_TOKEN,
  SPOTIFY_SECRET,
} from '@constants';
import type { Cache } from 'cache-manager';
import type { ISpotifyRenewAccessTokenResponse } from 'src/types/spotify-account';
import { Interval } from '@nestjs/schedule';
import { CommonResponseDto } from 'src/common/dto/common-response.dto';

@Injectable()
export class AuthService {
  private logger = new Logger(AuthService.name);
  constructor(
    @Inject(CACHE_MANAGER) private readonly cacheManager: Cache,
    private readonly config: ConfigService,
  ) {}

  /**
   * Renew Spotify Access Token
   */
  @Interval(ms('55 minutes'))
  async renewAccessToken(): Promise<CommonResponseDto> {
    try {
      const clientId = await this.config.get(SPOTIFY_CLIENT_ID);
      const secret = await this.config.get(SPOTIFY_SECRET);
      const basicToken = Buffer.from(clientId + ':' + secret).toString(
        'base64',
      );

      const refresh_token = await this.cacheManager.get(SPOTIFY_REFRESH_TOKEN);
      if (!refresh_token) {
        this.logger.warn('No refresh_token found.');
        return { success: false, message: 'No refresh_token found.' };
      }

      // Request Spotify Access Token Renewal
      const { data } = await axios<ISpotifyRenewAccessTokenResponse>({
        method: 'post',
        baseURL: SPOTIFY_ACCOUNT_URL,
        url: '/api/token',
        data: qs.stringify({
          grant_type: 'refresh_token',
          refresh_token,
        }),
        headers: {
          Authorization: `Basic ${basicToken}`,
          'Content-Type': 'application/x-www-form-urlencoded',
        },
      });

      // Store Renewed Access Token in Cache
      await this.cacheManager.set(
        SPOTIFY_ACCESS_TOKEN,
        data.access_token,
        ms('1 hour'),
      );
      this.logger.log('Renewed Access Token');
      return { success: true };
    } catch (err) {
      throw err;
    }
  }
}
