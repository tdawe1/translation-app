/**
 * Utility function for merging Tailwind CSS classes
 * Combines clsx and tailwind-merge functionality in a single dependency-free function
 */
export function cn(...classes: (string | undefined | null | false)[]): string {
  return classes.filter(Boolean).join(" ");
}
