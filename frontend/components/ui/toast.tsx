/**
 * Toast Component - Displays notification toasts
 *
 * Features:
 * - Auto-dismisses after 5 seconds
 * - Click to dismiss
 * - Smooth enter/exit animations
 * - Types: error (red), success (green), info (blue)
 */

"use client";

import { useEffect } from "react";
import { useToastStore } from "@/store/toast";

// Toast type styles matching the Data Factory design
const toastStyles: Record<
  string,
  { bg: string; border: string; icon: string }
> = {
  error: {
    bg: "bg-red-50",
    border: "border-red-200",
    icon: "✕",
  },
  success: {
    bg: "bg-green-50",
    border: "border-green-200",
    icon: "✓",
  },
  info: {
    bg: "bg-blue-50",
    border: "border-blue-200",
    icon: "ℹ",
  },
};

export function Toaster() {
  const { toasts, removeToast } = useToastStore();

  return (
    <div className="fixed bottom-6 right-6 z-50 flex flex-col gap-3 pointer-events-none">
      {toasts.map((toast) => (
        <Toast key={toast.id} toast={toast} onDismiss={removeToast} />
      ))}
    </div>
  );
}

interface ToastProps {
  toast: {
    id: string;
    type: "error" | "success" | "info";
    message: string;
  };
  onDismiss: (id: string) => void;
}

function Toast({ toast, onDismiss }: ToastProps) {
  const { bg, border, icon } = toastStyles[toast.type];

  useEffect(() => {
    // Play subtle sound for errors (optional, browser-dependent)
    if (toast.type === "error") {
      // Could add audio feedback here if desired
    }
  }, [toast.type]);

  return (
    <div
      className={`pointer-events-auto bento-card ${bg} ${border} px-4 py-3 min-w-[300px] max-w-md animate-in slide-in-from-bottom fade-in duration-300`}
      role="alert"
      aria-live={toast.type === "error" ? "assertive" : "polite"}
    >
      <div className="flex items-start gap-3">
        <span className="text-sm font-mono" aria-hidden="true">
          {icon}
        </span>
        <p className="flex-1 text-sm text-neutral-800">{toast.message}</p>
        <button
          onClick={() => onDismiss(toast.id)}
          className="text-neutral-400 hover:text-neutral-600 transition-colors"
          aria-label="Dismiss notification"
        >
          <span aria-hidden="true">×</span>
        </button>
      </div>
    </div>
  );
}
