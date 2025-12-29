/**
 * Modal - Reusable modal component following bento design language
 *
 * Accessibility features:
 * - role="dialog" and aria-modal="true" for screen readers
 * - aria-labelledby pointing to the title element
 * - aria-label on close button
 * - Escape key handling
 *
 * NOTE: For full focus trap support, consider migrating to Radix UI Dialog
 * which provides comprehensive focus management and keyboard navigation.
 */

import { useEffect, useRef } from "react";

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title?: string;
  children: React.ReactNode;
}

export function Modal({ isOpen, onClose, title, children }: ModalProps) {
  const overlayRef = useRef<HTMLDivElement>(null);
  const modalRef = useRef<HTMLDivElement>(null);
  const titleId = useRef<string>(`modal-title-${Math.random().toString(36).slice(2)}`);

  // Close on escape key
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };

    if (isOpen) {
      document.addEventListener("keydown", handleEscape);
      // Prevent body scroll
      document.body.style.overflow = "hidden";
      // Set focus to modal for accessibility
      modalRef.current?.focus();
    }

    return () => {
      document.removeEventListener("keydown", handleEscape);
      document.body.style.overflow = "";
    };
  }, [isOpen, onClose]);

  // Close on overlay click
  const handleOverlayClick = (e: React.MouseEvent) => {
    if (e.target === overlayRef.current) onClose();
  };

  if (!isOpen) return null;

  return (
    <div
      ref={overlayRef}
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      onClick={handleOverlayClick}
    >
      <div
        ref={modalRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={title ? titleId.current : undefined}
        tabIndex={-1}
        className="w-full max-w-md bg-white border border-neutral-200 shadow-lg outline-none"
      >
        {title && (
          <div className="px-6 py-4 border-b border-neutral-200 flex items-center justify-between">
            <h3 id={titleId.current} className="text-lg font-light tracking-tighter">
              {title}
            </h3>
            <button
              onClick={onClose}
              aria-label="Close dialog"
              className="ml-4 text-neutral-400 hover:text-neutral-600 transition-colors"
            >
              <svg
                width="20"
                height="20"
                viewBox="0 0 20 20"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.5"
              >
                <path d="M4 4L16 16M4 16L16 4" />
              </svg>
            </button>
          </div>
        )}
        <div className="px-6 py-4">{children}</div>
      </div>
    </div>
  );
}
