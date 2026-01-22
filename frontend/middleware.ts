import createMiddleware from 'next-intl/middleware';

const locales = ['en', 'es', 'fr', 'de', 'ja'] as const;

export default createMiddleware({
  locales,
  defaultLocale: 'en',
  localePrefix: 'never'
});

export const config = {
  matcher: ['/((?!api|_next|_vercel|.*\\..*).*)']
};
