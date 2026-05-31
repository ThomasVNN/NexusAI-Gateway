import { TranslationKeys } from './en';
import { vi } from './vi';

export type Language = 'en' | 'vi' | 'zh';

type TranslationDict = {
  [key in Language]: TranslationKeys;
};

const translations: TranslationDict = {
  en,
  vi,
  zh: en,
};

export function getTranslations(lang: Language): TranslationKeys {
  return translations[lang] || translations.en;
}

export { en, vi };
export type { TranslationKeys };
