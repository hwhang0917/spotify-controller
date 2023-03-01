import open from 'open';
import { CacheModule, Module, OnModuleInit } from '@nestjs/common';
import { ConfigModule } from '@nestjs/config';
import { ScheduleModule } from '@nestjs/schedule';
import { DEFAULT_PORT } from '@constants';
import { validationSchema } from '@config';
import { AppController } from './app.controller';
import { SpotifyModule } from './spotify/spotify.module';
import { AuthModule } from './auth/auth.module';

@Module({
  imports: [
    ConfigModule.forRoot({ validationSchema, isGlobal: true }),
    CacheModule.register({ isGlobal: true }),
    ScheduleModule.forRoot(),
    SpotifyModule,
    AuthModule,
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
