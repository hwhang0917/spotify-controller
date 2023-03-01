import Joi from 'joi';
import { SPOTIFY_CLIENT_ID, SPOTIFY_SECRET } from '@constants';

/**
 * Dotenv Validation Schema
 */
export const validationSchema = Joi.object({
  [SPOTIFY_CLIENT_ID]: Joi.string().required(),
  [SPOTIFY_SECRET]: Joi.string().required(),
});
