import { join } from 'path';
import { readFile } from 'fs/promises';
import qs from 'qs';
import ms from 'ms';
import axios from 'axios';
import {
  BadRequestException,
  CACHE_MANAGER,
  Controller,
  Get,
  Inject,
  Query,
  Res,
} from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import {
  REDIRECT_URI,
  SPOTIFY_ACCESS_TOKEN,
  SPOTIFY_ACCOUNT_URL,
  SPOTIFY_CLIENT_ID,
  SPOTIFY_REFRESH_TOKEN,
  SPOTIFY_SCOPE,
  SPOTIFY_SECRET,
  SPOTIFY_STATE,
} from '@constants';
import { generateRandomHash } from '@utils/random';
import type { Response } from 'express';
import type { Cache } from 'cache-manager';
import type { ISpotifyAccessTokenResponse } from './types/spotify-account';

@Controller()
export class AppController {
  constructor(
    @Inject(CACHE_MANAGER) private readonly cacheManager: Cache,
    private readonly config: ConfigService,
  ) {}

  @Get()
  async spotifyLogin(@Res({ passthrough: true }) res: Response) {
    // Get Client ID and State
    const client_id = await this.config.get(SPOTIFY_CLIENT_ID);
    const state = generateRandomHash();

    // Store state to In-Memory Cache
    await this.cacheManager.set(SPOTIFY_STATE, state, ms('15 minutes'));

    const spotifyLoginUrl = new URL(
      '/authorize?' +
        qs.stringify({
          response_type: 'code',
          client_id,
          scope: SPOTIFY_SCOPE,
          redirect_uri: REDIRECT_URI,
          state,
        }),
      SPOTIFY_ACCOUNT_URL,
    );

    res.redirect(spotifyLoginUrl.href);
  }

  @Get('callback')
  async spotifyCallback(
    @Query('code') code: string,
    @Query('state') state: string,
    @Res({ passthrough: true }) res: Response,
  ) {
    try {
      if (!code || !state) {
        throw new BadRequestException('invalid_request');
      }

      const storedState = await this.cacheManager.get(SPOTIFY_STATE);
      if (storedState !== state) {
        throw new BadRequestException('state_mismatch');
      }

      const clientId = await this.config.get(SPOTIFY_CLIENT_ID);
      const secret = await this.config.get(SPOTIFY_SECRET);
      const basicToken = Buffer.from(clientId + ':' + secret).toString(
        'base64',
      );

      // Request Spotify Access Token
      const { data } = await axios<ISpotifyAccessTokenResponse>({
        method: 'post',
        baseURL: SPOTIFY_ACCOUNT_URL,
        url: '/api/token',
        data: qs.stringify({
          code,
          redirect_uri: REDIRECT_URI,
          grant_type: 'authorization_code',
        }),
        headers: {
          Authorization: `Basic ${basicToken}`,
          'Content-Type': 'application/x-www-form-urlencoded',
        },
      });

      // Store Access/Refresh Token in Cache
      await this.cacheManager.set(
        SPOTIFY_ACCESS_TOKEN,
        data.access_token,
        ms('1 hour'),
      );
      await this.cacheManager.set(SPOTIFY_REFRESH_TOKEN, data.refresh_token, 0);

      // render static HTML
      const staticHtml = await readFile(
        join(__dirname, 'static', 'index.html'),
        {
          encoding: 'utf8',
        },
      );
      res.send(staticHtml);
    } catch (err) {
      throw err;
    }
  }
}
