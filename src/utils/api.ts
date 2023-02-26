import { SPOTIFY_API_URL } from '@constants';
import axios from 'axios';

export const api = axios.create({
  baseURL: SPOTIFY_API_URL,
  params: {
    locale: 'ko_KR',
  },
  headers: {
    'Content-Type': 'application/json',
  },
});
