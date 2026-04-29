/**
 * WatcherConfigForm - Form for editing watcher configuration
 */

import { useState, useEffect } from "react";
import { useWatcherStore } from "@/store/watcher";
import { requestJobAlertPermission } from "@/lib/job-alerts";
import type { WatcherConfig } from "@/lib/api";

interface WatcherConfigFormProps {
  onClose?: () => void;
}

export function WatcherConfigForm({ onClose }: WatcherConfigFormProps) {
  const { config, updateConfig, configLoading } = useWatcherStore();

  // Local form state
  const [formData, setFormData] = useState<Partial<WatcherConfig>>(
    config || {
      rss_feed_url: "",
      websocket_enabled: true,
      min_reward: 0,
      max_reward: 999,
      included_language_pairs: [],
      enable_desktop_notifications: false,
      enable_sound_notifications: false,
      enable_email_notifications: false,
      auto_accept_enabled: false,
    }
  );

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Update form data when config loads
  useEffect(() => {
    if (config) {
      setFormData(config);
    }
  }, [config]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setError(null);

    try {
      await updateConfig(formData);
      onClose?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update configuration");
    } finally {
      setSubmitting(false);
    }
  };

  const handleChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>
  ) => {
    const { name, value, type } = e.target;
    const checked = type === "checkbox" ? (e.target as HTMLInputElement).checked : false;

    if (name === "enable_desktop_notifications" && checked) {
      void requestJobAlertPermission().then((permission) => {
        if (!mountedRef.current) return;
        if (permission === "denied") {
          setNotice("Browser notifications are blocked. Job pages will still auto-open if pop-ups are allowed for this site.");
        } else if (permission === "unsupported") {
          setNotice("This browser does not support desktop notifications. Job pages will still auto-open if pop-ups are allowed for this site.");
        } else {
          setNotice(null);
        }
      });
    }
    if (name === "enable_desktop_notifications" && !checked) {
      setNotice(null);
    }

    setFormData((prev) => ({
      ...prev,
      [name]:
        type === "checkbox"
          ? checked
          : type === "number"
            ? parseFloat(value) || 0
            : value,
    }));
  };

  const errorId = "config-form-error";

  return (
    <form onSubmit={handleSubmit} className="space-y-4" aria-labelledby="config-form-title">
      {/* Hidden title for screen readers */}
      <h2 id="config-form-title" className="sr-only">
        Watcher Configuration
      </h2>

      {error && (
        <div
          id={errorId}
          className="p-3 bg-red-50 border border-red-200 text-red-700 text-sm"
          role="alert"
          aria-live="assertive"
        >
          {error}
        </div>
      )}

      {/* RSS Feed URL */}
      <div>
        <label
          htmlFor="rss-feed-url"
          className="block font-mono text-xs uppercase tracking-widest text-neutral-500 mb-1"
        >
          RSS Feed URL
        </label>
        <input
          id="rss-feed-url"
          type="url"
          name="rss_feed_url"
          value={formData.rss_feed_url || ""}
          onChange={handleChange}
          required
          className="w-full px-3 py-2 border border-neutral-200 focus:border-blue-600 focus:outline-none text-sm font-mono"
          placeholder="https://example.com/feed.xml"
          aria-describedby={error ? errorId : undefined}
        />
      </div>

      {/* Reward Range */}
      <div className="grid grid-cols-2 gap-4">
        <div>
          <label
            htmlFor="min-reward"
            className="block font-mono text-xs uppercase tracking-widest text-neutral-500 mb-1"
          >
            Min Reward ($)
          </label>
          <input
            id="min-reward"
            type="number"
            name="min_reward"
            value={formData.min_reward || 0}
            onChange={handleChange}
            min="0"
            step="0.01"
            required
            className="w-full px-3 py-2 border border-neutral-200 focus:border-blue-600 focus:outline-none text-sm font-mono"
            aria-describedby={error ? errorId : undefined}
          />
        </div>
        <div>
          <label
            htmlFor="max-reward"
            className="block font-mono text-xs uppercase tracking-widest text-neutral-500 mb-1"
          >
            Max Reward ($)
          </label>
          <input
            id="max-reward"
            type="number"
            name="max_reward"
            value={formData.max_reward || 999}
            onChange={handleChange}
            min="0"
            step="0.01"
            required
            className="w-full px-3 py-2 border border-neutral-200 focus:border-blue-600 focus:outline-none text-sm font-mono"
            aria-describedby={error ? errorId : undefined}
          />
        </div>
      </div>

      {/* Toggles */}
      <div className="space-y-3 pt-2" role="group" aria-label="Notification settings">
        {/* WebSocket Monitoring */}
        <label className="flex items-center justify-between cursor-pointer">
          <span className="text-sm">Enable WebSocket Monitoring</span>
          <div className="relative">
            <input
              type="checkbox"
              name="websocket_enabled"
              id="websocket-enabled"
              checked={formData.websocket_enabled || false}
              onChange={handleChange}
              className="sr-only peer"
            />
            <div className="w-10 h-5 bg-neutral-300 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-blue-600" />
          </div>
        </label>

        {/* Desktop Notifications */}
        <label className="flex items-center justify-between cursor-pointer">
          <span className="text-sm">Desktop Notifications</span>
          <div className="relative">
            <input
              type="checkbox"
              name="enable_desktop_notifications"
              id="desktop-notifications"
              checked={formData.enable_desktop_notifications || false}
              onChange={handleChange}
              className="sr-only peer"
            />
            <div className="w-10 h-5 bg-neutral-300 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-blue-600" />
          </div>
        </label>

        {/* Sound Notifications */}
        <label className="flex items-center justify-between cursor-pointer">
          <span className="text-sm">Sound Notifications</span>
          <div className="relative">
            <input
              type="checkbox"
              name="enable_sound_notifications"
              id="sound-notifications"
              checked={formData.enable_sound_notifications || false}
              onChange={handleChange}
              className="sr-only peer"
            />
            <div className="w-10 h-5 bg-neutral-300 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-blue-600" />
          </div>
        </label>

        {/* Email Notifications */}
        <label className="flex items-center justify-between cursor-pointer">
          <span className="text-sm">Email Notifications</span>
          <div className="relative">
            <input
              type="checkbox"
              name="enable_email_notifications"
              id="email-notifications"
              checked={formData.enable_email_notifications || false}
              onChange={handleChange}
              className="sr-only peer"
            />
            <div className="w-10 h-5 bg-neutral-300 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-blue-600" />
          </div>
        </label>
      </div>

      {/* Actions */}
      <div className="flex gap-3 pt-4">
        <button
          type="submit"
          disabled={submitting || configLoading}
          className="flex-1 py-2 bg-neutral-900 text-white text-sm transition-colors duration-150 hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed"
          aria-describedby={error ? errorId : undefined}
        >
          {submitting ? "Saving..." : "Save Configuration"}
        </button>
        <button
          type="button"
          onClick={onClose}
          disabled={submitting}
          className="px-6 py-2 border border-neutral-300 text-sm transition-colors duration-150 hover:border-neutral-400 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          Cancel
        </button>
      </div>
    </form>
  );
}
