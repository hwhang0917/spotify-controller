import { DEFAULT_LOCALE } from '@constants';

/**
 * Get locale from environmental variables
 */
const getLocaleEnv = () => {
  return (
    process.env.LC_ALL ||
    process.env.LC_MESSAGES ||
    process.env.LANG ||
    process.env.LANGUAGE
  );
};

/**
 * Parse ISO 639 from locale
 */
export const getLanguageLocale = () => {
  return getLocaleEnv().split('.')?.at(0) ?? DEFAULT_LOCALE;
};
