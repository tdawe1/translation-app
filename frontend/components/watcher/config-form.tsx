/**
 * WatcherConfigForm - Form for editing watcher configuration
 */

import { useState } from "react";
import { useWatcherStore } from "@/store/watcher";
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
  if (config && JSON.stringify(formData) !== JSON.stringify(config)) {
    setFormData(config);
  }

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
    setFormData((prev) => ({
      ...prev,
      [name]:
        type === "checkbox"
          ? (e.target as HTMLInputElement).checked
          : type === "number"
            ? parseFloat(value) || 0
            : value,
    }));
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {error && (
        <div className="p-3 bg-red-50 border border-red-200 text-red-700 text-sm">
          {error}
        </div>
      )}

      {/* RSS Feed URL */}
      <div>
        <label className="block font-mono text-xs uppercase tracking-widest text-neutral-500 mb-1">
          RSS Feed URL
        </label>
        <input
          type="url"
          name="rss_feed_url"
          value={formData.rss_feed_url || ""}
          onChange={handleChange}
          required
          className="w-full px-3 py-2 border border-neutral-200 focus:border-blue-600 focus:outline-none text-sm font-mono"
          placeholder="https://example.com/feed.xml"
        />
      </div>

      {/* Reward Range */}
      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block font-mono text-xs uppercase tracking-widest text-neutral-500 mb-1">
            Min Reward ($)
          </label>
          <input
            type="number"
            name="min_reward"
            value={formData.min_reward || 0}
            onChange={handleChange}
            min="0"
            step="0.01"
            required
            className="w-full px-3 py-2 border border-neutral-200 focus:border-blue-600 focus:outline-none text-sm font-mono"
          />
        </div>
        <div>
          <label className="block font-mono text-xs uppercase tracking-widest text-neutral-500 mb-1">
            Max Reward ($)
          </label>
          <input
            type="number"
            name="max_reward"
            value={formData.max_reward || 999}
            onChange={handleChange}
            min="0"
            step="0.01"
            required
            className="w-full px-3 py-2 border border-neutral-200 focus:border-blue-600 focus:outline-none text-sm font-mono"
          />
        </div>
      </div>

      {/* Toggles */}
      <div className="space-y-3 pt-2">
        {/* WebSocket Monitoring */}
        <label className="flex items-center justify-between cursor-pointer">
          <span className="text-sm">Enable WebSocket Monitoring</span>
          <div className="relative">
            <input
              type="checkbox"
              name="websocket_enabled"
              checked={formData.websocket_enabled || false}
              onChange={handleChange}
              className="sr-only peer"
            />
            <div className="w-10 h-5 bg-neutral-300 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-blue-600" />
          </div>
        </label>

        {/* Auto Accept */}
        <label className="flex items-center justify-between cursor-pointer">
          <span className="text-sm">Auto Accept Jobs</span>
          <div className="relative">
            <input
              type="checkbox"
              name="auto_accept_enabled"
              checked={formData.auto_accept_enabled || false}
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
