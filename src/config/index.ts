import Joi from 'joi';

export const validationSchema = Joi.object({
  SPOTIFY_CLIENT_ID: Joi.string().required(),
  SPOTIFY_SECRET: Joi.string().required(),
});
