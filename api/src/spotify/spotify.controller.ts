import {
  BadRequestException,
  Controller,
  Get,
  ParseIntPipe,
  Post,
  Query,
} from '@nestjs/common';
import { SpotifyService } from './spotify.service';

@Controller('spotify')
export class SpotifyController {
  constructor(private readonly spotifyService: SpotifyService) {}

  @Get('search')
  async searchSongs(
    @Query('q', {
      transform: (value) => {
        if (!value)
          throw new BadRequestException('Search query `q` is required.');
        return value;
      },
    })
    q: string,
    @Query('page', {
      transform: (value) => {
        return !value ? undefined : Number(value);
      },
    })
    page: number,
  ) {
    return this.spotifyService.searchSongs(q, page);
  }

  @Get('current')
  async getCurrentlyPlayingTrack() {
    return this.spotifyService.getCurrentlyPlayingTrack();
  }

  @Get('queue')
  async getQueue() {
    return this.spotifyService.getQueue();
  }

  @Post('queue')
  async addTrackToQueue(@Query('id') trackId: string) {
    return this.spotifyService.addTrackToQueue(trackId);
  }
}