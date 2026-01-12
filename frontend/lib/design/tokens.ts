/**
 * Design Token Constants - Data Factory Design Language
 *
 * Single source of truth for all visual design values.
 * Import these constants instead of hardcoding values.
 */

// ============================================================================
// COLOR TOKENS
// ============================================================================

export const COLORS = {
  // Border colors
  border: {
    default: "rgb(200, 200, 200)", // bento-card default
    divider: "border-neutral-200",
    strong: "border-neutral-300",
  },

  // Text colors
  text: {
    primary: "text-neutral-900",
    secondary: "text-neutral-600",
    meta: "text-neutral-500",
    disabled: "text-neutral-400",
  },

  // Background colors
  bg: {
    page: "bg-neutral-50",
    card: "bg-white",
  },

  // ROYGBIV accent colors (for headings/labels ONLY)
  accent: {
    red: "text-red-600",
    orange: "text-orange-600",
    yellow: "text-yellow-600",
    green: "text-green-600",
    cyan: "text-cyan-600",
    blue: "text-blue-600",
    indigo: "text-indigo-600",
    violet: "text-violet-600",
  },

  // Semantic colors
  semantic: {
    error: "bg-red-50 border-red-200 text-red-700",
    success: "bg-green-50 border-green-200 text-green-800",
    warning: "bg-yellow-50 border-yellow-200 text-yellow-800",
    info: "bg-blue-50 border-blue-200 text-blue-800",
  },
} as const;

// ============================================================================
// SPACING TOKENS
// ============================================================================

export const SPACING = {
  // Card padding variants
  card: {
    sm: "p-4",
    md: "p-6",
    lg: "p-8",
    xl: "p-12",
  },

  // Section spacing
  section: "py-24",
  hero: "pt-44",

  // Gap between elements
  gap: {
    sm: "gap-2",
    md: "gap-4",
    lg: "gap-6",
    xl: "gap-8",
  },
} as const;

// ============================================================================
// ANIMATION TOKENS
// ============================================================================

export const ANIMATION = {
  fade: "animate-fade-in",
  duration: "duration-150",
  easing: "ease",
  staggerDelay: 25, // ms between staggered animations
  maxStaggerIndex: 4, // max stagger index (0-4)
} as const;

// ============================================================================
// TYPOGRAPHY TOKENS
// ============================================================================

export const TYPOGRAPHY = {
  // Font families
  font: {
    display: "IBM Plex Sans",
    mono: "IBM Plex Mono",
  },

  // Heading sizes
  heading: {
    hero: "text-6xl",
    section: "text-4xl",
    card: "text-2xl",
    page: "text-xl",
  },

  // Font weights
  weight: {
    light: "font-light",
    normal: "font-normal",
    medium: "font-medium",
  },

  // Label pattern (uppercase tracking)
  label: "font-mono text-xs uppercase tracking-widest",
} as const;

// ============================================================================
// DESIGN TOKENS EXPORT
// ============================================================================

export const DESIGN = {
  colors: COLORS,
  spacing: SPACING,
  animation: ANIMATION,
  typography: TYPOGRAPHY,

  /**
   * Get cyclic accent color by index
   * @param index - Number to cycle through ROYGBIV colors
   * @returns Tailwind class string for accent color
   */
  getAccentColor(index: number): string {
    const accents: Array<keyof typeof COLORS.accent> = [
      "red",
      "orange",
      "yellow",
      "green",
      "cyan",
      "blue",
      "indigo",
      "violet",
    ];
    const key = accents[index % accents.length];
    return COLORS.accent[key];
  },

  /**
   * Get stagger animation delay in milliseconds
   * @param index - Stagger index (0-4)
   * @returns CSS animation delay value in milliseconds
   */
  getStaggerDelay(index: number): string {
    const clampedIndex = Math.min(index, ANIMATION.maxStaggerIndex);
    return `${clampedIndex * ANIMATION.staggerDelay}ms`;
  },
} as const;

// ============================================================================
// TYPE EXPORTS
// ============================================================================

export type AccentColor = keyof typeof COLORS.accent;
export type CardSize = keyof typeof SPACING.card;
export type GapSize = keyof typeof SPACING.gap;
