import {getRequestConfig} from 'next-intl/server';

type Locale = 'en' | 'es' | 'fr' | 'de' | 'ja';

const locales: Locale[] = ['en', 'es', 'fr', 'de', 'ja'];

export default getRequestConfig(async ({requestLocale}) => {
  let locale = await requestLocale;

  if (!locale || !locales.includes(locale as Locale)) {
    locale = 'en';
  }

  return {
    locale,
    messages: (await import(`../messages/${locale}.json`)).default
  };
});
