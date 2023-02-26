import { join } from 'path';
import { NestFactory } from '@nestjs/core';
import { AppModule } from './app.module';
import { DEFAULT_PORT } from '@constants';
import type { NestExpressApplication } from '@nestjs/platform-express';

async function bootstrap() {
  const app = await NestFactory.create<NestExpressApplication>(AppModule);

  // Serve Static files in public
  app.useStaticAssets(join(__dirname, '..', 'public'));
  await app.listen(DEFAULT_PORT);
}
bootstrap();
