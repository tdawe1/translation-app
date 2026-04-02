import { notFound } from "next/navigation";
import type { ReactNode } from "react";

type Locale = "en" | "es" | "fr" | "de" | "ja";

const locales: Locale[] = ["en", "es", "fr", "de", "ja"];

export default async function LocaleLayout({
  children,
  params,
}: {
  children: ReactNode;
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;

  if (!locales.includes(locale as Locale)) {
    notFound();
  }

  return children;
}
