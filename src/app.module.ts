import { CacheModule, Module } from '@nestjs/common';
import { ConfigModule } from '@nestjs/config';
import { AppController } from './app.controller';
import { AppService } from './app.service';
import { validationSchema } from '@config';

@Module({
  imports: [ConfigModule.forRoot({ validationSchema }), CacheModule.register()],
  controllers: [AppController],
  providers: [AppService],
})
export class AppModule {}
