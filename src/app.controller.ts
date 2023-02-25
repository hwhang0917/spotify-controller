import * as qs from 'qs';
import { Controller, Get, Headers, Req, Res } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { AppService } from './app.service';
import { SPOTIFY_CLIENT_ID } from '@constants';
import { generateRandomHash } from '@utils/random';
import type { Request, Response } from 'express';

@Controller()
export class AppController {
  constructor(
    private readonly appService: AppService,
    private readonly config: ConfigService,
  ) {}

  @Get('login')
  async spotifyLogin(
    @Req() req: Request,
    @Res({ passthrough: true }) res: Response,
  ) {
    const client_id = await this.config.get(SPOTIFY_CLIENT_ID);
    const state = generateRandomHash();
    const backendUrl = `${req.protocol}://${req.get('host')}`;
    const { href: redirect_uri } = new URL('/callback', backendUrl);

    res.redirect(
      `https://accounts.spotify.com/authorize?` +
        qs.stringify({
          response_type: 'code',
          client_id,
          scope: 'user-modify-playback-state',
          redirect_uri,
          state,
        }),
    );
  }
}
