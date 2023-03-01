import { getLanguageLocale } from './locale';

const formatLang = getLanguageLocale().split('_').at(0);
export const listFormatter = new Intl.ListFormat(formatLang);
