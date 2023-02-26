import axios from 'axios';
import { SPOTIFY_API_URL } from '@constants';
import { getLanguageLocale } from './locale';

export const api = axios.create({
  baseURL: SPOTIFY_API_URL,
  params: {
    locale: getLanguageLocale(),
  },
  headers: {
    'Content-Type': 'application/json',
  },
});
