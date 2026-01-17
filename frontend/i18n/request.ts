import {notFound} from 'next/navigation';
import {getRequestConfig} from 'next-intl/server';

type Locale = 'en' | 'es' | 'fr' | 'de';

const locales: Locale[] = ['en', 'es', 'fr', 'de'];

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
