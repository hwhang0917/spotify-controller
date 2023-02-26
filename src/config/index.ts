import { LOCALE_REGEX } from '@constants';
import Joi from 'joi';

/**
 * Dotenv Validation Schema
 */
export const validationSchema = Joi.object({
  SPOTIFY_CLIENT_ID: Joi.string().required(),
  SPOTIFY_SECRET: Joi.string().required(),
  LOCALE: Joi.string().required().regex(LOCALE_REGEX),
});
