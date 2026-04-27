import '@testing-library/jest-dom';
import { vi } from 'vitest';

const i18nMock = vi.hoisted(() => {
  const messages = {
    auth: {
      email: 'Email',
      password: 'Password',
      signIn: 'Sign in',
      signUp: 'Create account',
      signingIn: 'Signing in',
      creatingAccount: 'Creating account',
      magicLink: 'Send magic link instead',
      passwordTooShort: 'Password must be at least 8 characters',
      passwordMismatch: 'Passwords do not match',
      signInWithGoogle: 'Continue with Google',
      signInWithGitHub: 'Continue with GitHub',
    },
    common: {
      confirm: 'Confirm',
      api: {
        error: 'An unexpected error occurred',
      },
    },
  } as const;

  function resolveMessage(namespace: string | undefined, key: string): string {
    const root = namespace && namespace in messages
      ? messages[namespace as keyof typeof messages]
      : messages;
    const value = key.split('.').reduce<unknown>((current, part) => {
      if (current && typeof current === 'object' && part in current) {
        return (current as Record<string, unknown>)[part];
      }
      return undefined;
    }, root);

    return typeof value === 'string' ? value : key;
  }

  return { resolveMessage };
});

vi.mock('next-intl', () => ({
  NextIntlClientProvider: ({ children }: { children: unknown }) => children,
  useTranslations: (namespace?: string) => (key: string) =>
    i18nMock.resolveMessage(namespace, key),
}));
