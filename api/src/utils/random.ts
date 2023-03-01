import { randomBytes } from 'crypto';

/**
 * 랜덤 해시값 생성
 * @param byte 바이트값
 */
export const generateRandomHash = (byte = 5) =>
  randomBytes(byte).toString('hex');
