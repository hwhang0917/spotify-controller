import { NestFactory } from '@nestjs/core';
import { AppModule } from './app.module';
import { DEFAULT_PORT } from '@constants';

async function bootstrap() {
  const app = await NestFactory.create(AppModule);
  await app.listen(DEFAULT_PORT);
}
bootstrap();
