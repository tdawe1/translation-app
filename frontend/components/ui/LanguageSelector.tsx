"use client";

import { useState, useRef, useEffect } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { localeFlags, localeNames, type Locale } from '@/i18n/config';
import { cn } from '@/lib/utils';

export function LanguageSelector() {
  const [isOpen, setIsOpen] = useState(false);
  const pathname = usePathname();
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Extract current locale from pathname
  // Path structure is usually /[locale]/...
  const segments = pathname?.split('/') || [];
  const currentLocaleCode = segments[1] as Locale;
  
  // Validate current locale, fallback to 'en'
  const currentLocale: Locale = localeNames[currentLocaleCode] 
    ? currentLocaleCode 
    : 'en';

  // Handle click outside to close dropdown
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  // Generate URL for a target locale
  const getTargetUrl = (locale: Locale) => {
    if (!pathname) return `/${locale}`;
    const segments = pathname.split('/');
    // Replace the locale segment (index 1)
    segments[1] = locale;
    return segments.join('/') || '/';
  };

  return (
    <div className="relative" ref={dropdownRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className={cn(
          "flex items-center gap-2 px-3 py-2 text-sm font-medium bg-white",
          "border border-neutral-200 rounded-md",
          "hover:border-neutral-300 transition-colors duration-150",
          "focus:outline-none focus:ring-2 focus:ring-blue-600 focus:ring-offset-1",
          isOpen && "border-neutral-300 bg-neutral-50"
        )}
        aria-expanded={isOpen}
        aria-haspopup="true"
        aria-label="Select language"
      >
        <span className="text-lg leading-none" aria-hidden="true">
          {localeFlags[currentLocale]}
        </span>
        <span className="hidden sm:inline-block text-neutral-700">
          {localeNames[currentLocale]}
        </span>
        <svg
          className={cn(
            "w-4 h-4 text-neutral-500 transition-transform duration-200",
            isOpen && "rotate-180"
          )}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {isOpen && (
        <div 
          className="absolute right-0 z-50 w-48 py-1 mt-2 bg-white border border-neutral-200 rounded-lg shadow-lg origin-top-right animate-in fade-in zoom-in-95 duration-100"
          role="menu"
          aria-orientation="vertical"
        >
          {(Object.keys(localeNames) as Locale[]).map((locale) => (
            <Link
              key={locale}
              href={getTargetUrl(locale)}
              className={cn(
                "flex items-center gap-3 px-4 py-2.5 text-sm transition-colors w-full",
                "hover:bg-neutral-50 focus:bg-neutral-50 focus:outline-none",
                currentLocale === locale 
                  ? "bg-blue-50/50 text-blue-700 font-medium" 
                  : "text-neutral-700"
              )}
              onClick={() => setIsOpen(false)}
              role="menuitem"
              aria-current={currentLocale === locale ? "true" : undefined}
            >
              <span className="text-lg leading-none" aria-hidden="true">
                {localeFlags[locale]}
              </span>
              <span>{localeNames[locale]}</span>
              {currentLocale === locale && (
                <svg className="w-4 h-4 ml-auto text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
              )}
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
