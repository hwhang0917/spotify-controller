import open from 'open';
import { CacheModule, Module, OnModuleInit } from '@nestjs/common';
import { ConfigModule } from '@nestjs/config';
import { AppController } from './app.controller';
import { validationSchema } from '@config';
import { DEFAULT_PORT } from '@constants';
import { SpotifyModule } from './spotify/spotify.module';

@Module({
  imports: [
    ConfigModule.forRoot({ validationSchema, isGlobal: true }),
    CacheModule.register({ isGlobal: true }),
    SpotifyModule,
  ],
  controllers: [AppController],
})
export class AppModule implements OnModuleInit {
  async onModuleInit() {
    // Open OAuth login on module init
    const baseDomain = `http://localhost:${DEFAULT_PORT}`;
    await open(baseDomain);
  }
}
