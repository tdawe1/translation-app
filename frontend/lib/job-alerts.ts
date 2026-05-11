import type { Job } from "@/store/jobs";

export type JobAlertPayload = Pick<
  Job,
  "id" | "title" | "reward" | "url" | "source" | "currency"
>;

export interface JobAlertResult {
  opened: boolean;
  notified: boolean;
  blocked: boolean;
  reason?: string;
  safeUrl?: string;
}

export interface JobAlertWindowState {
  browser_process_alive: boolean;
  current_url?: string;
  current_title?: string;
  current_action_step?: string;
  current_job_id?: string;
  logged_in_state: "unknown";
  devtools_connected: false;
}

let preparedJobWindow: Window | null = null;
let sharedAudioContext: AudioContext | null = null;
let fallbackAudioElement: HTMLAudioElement | null = null;
let alertToneDataUrl: string | null = null;
const JOB_WINDOW_NAME = "gengowatcher-job-window";
let lastKnownJobWindowState: JobAlertWindowState = {
  browser_process_alive: false,
  logged_in_state: "unknown",
  devtools_connected: false,
};

export function closePreparedJobAlertWindow(): void {
  if (
    preparedJobWindow &&
    !preparedJobWindow.closed &&
    typeof preparedJobWindow.close === "function"
  ) {
    preparedJobWindow.close();
  }
  preparedJobWindow = null;
  lastKnownJobWindowState = {
    ...lastKnownJobWindowState,
    browser_process_alive: false,
    current_action_step: "Dashboard job alert tab closed",
  };
}

const PRIVATE_IPV4_RANGES = [
  /^10\./,
  /^127\./,
  /^169\.254\./,
  /^172\.(1[6-9]|2\d|3[0-1])\./,
  /^192\.168\./,
  /^0\./,
];

export function canUseBrowserNotifications(): boolean {
  return typeof window !== "undefined" && "Notification" in window;
}

export async function requestJobAlertPermission(): Promise<
  NotificationPermission | "unsupported"
> {
  if (!canUseBrowserNotifications()) {
    return "unsupported";
  }

  if (Notification.permission === "default") {
    return Notification.requestPermission();
  }

  return Notification.permission;
}

export function getSafeJobUrl(rawUrl: string): string | null {
  if (typeof window === "undefined") {
    return null;
  }

  try {
    const parsed = new URL(rawUrl, window.location.origin);
    const host = parsed.hostname.toLowerCase();

    if (parsed.protocol !== "https:" && parsed.protocol !== "http:") {
      return null;
    }

    if (
      host === "localhost" ||
      host.endsWith(".localhost") ||
      host.endsWith(".local") ||
      host === "::1" ||
      PRIVATE_IPV4_RANGES.some((range) => range.test(host))
    ) {
      return null;
    }

    return parsed.toString();
  } catch {
    return null;
  }
}

export function formatJobSource(source: string): string {
  const normalized = source.trim().toLowerCase();
  switch (normalized) {
    case "rss":
      return "RSS";
    case "websocket":
    case "gengo_ws":
    case "gengo-websocket":
      return "WebSocket";
    case "external":
      return "External";
    default:
      return source.trim() || "Job";
  }
}

export function prepareJobAlertWindow(): boolean {
  if (typeof window === "undefined") {
    return false;
  }

  const jobWindow = window.open("about:blank", JOB_WINDOW_NAME);
  if (!jobWindow) {
    return false;
  }

  preparedJobWindow = jobWindow;
  lastKnownJobWindowState = {
    browser_process_alive: true,
    current_title: "GengoWatcher Job Window",
    current_action_step: "Dashboard job alert tab armed",
    logged_in_state: "unknown",
    devtools_connected: false,
  };
  try {
    jobWindow.opener = null;
    jobWindow.document.title = "GengoWatcher Job Window";
    jobWindow.document.body.innerHTML = [
      '<main style="font-family: sans-serif; padding: 2rem; line-height: 1.5;">',
      '<h1 style="font-size: 1.25rem; margin: 0 0 0.5rem;">GengoWatcher is armed</h1>',
      '<p style="margin: 0; color: #555;">Detected jobs will open here automatically.</p>',
      "</main>",
    ].join("");
  } catch {
    // Some browsers restrict about:blank document access; the window reference still works.
  }

  return true;
}

export function getPreparedJobAlertWindowState(): JobAlertWindowState {
  let alive = false;
  if (preparedJobWindow) {
    try {
      alive = !preparedJobWindow.closed;
    } catch {
      alive = false;
    }
  }

  lastKnownJobWindowState = {
    ...lastKnownJobWindowState,
    browser_process_alive: alive,
  };
  return { ...lastKnownJobWindowState };
}

function clearWindowOpener(target: Window): void {
  try {
    target.opener = null;
  } catch {
    // Reused job windows may already be cross-origin after navigating to gengo.com.
  }
}

function focusWindow(target: Window): void {
  try {
    target.focus();
  } catch {
    // Focusing can be denied by the browser or window manager.
  }
}

function navigateWindow(target: Window, url: string): Window | null {
  try {
    target.location.href = url;
    return target;
  } catch {
    // Once the prepared tab has navigated to gengo.com, direct Location access can
    // throw. Re-target the named tab without touching cross-origin properties.
  }

  try {
    return window.open(url, JOB_WINDOW_NAME);
  } catch {
    return null;
  }
}

function openJobWindow(url: string): Window | null {
  const openedWindow = window.open("about:blank", JOB_WINDOW_NAME);
  if (!openedWindow) {
    return null;
  }

  clearWindowOpener(openedWindow);
  const navigatedWindow = navigateWindow(openedWindow, url);
  lastKnownJobWindowState = {
    ...lastKnownJobWindowState,
    browser_process_alive: Boolean(navigatedWindow),
    current_url: navigatedWindow ? url : lastKnownJobWindowState.current_url,
  };
  if (!navigatedWindow) {
    try {
      openedWindow.close();
    } catch {
      // The blank tab may already be inaccessible or closed by the browser.
    }
  }
  return navigatedWindow;
}

function formatReward(job: JobAlertPayload): string {
  const currency = job.currency || "USD";
  if (currency.toUpperCase() === "USD") {
    return `$${job.reward.toFixed(2)}`;
  }
  return `${job.reward.toFixed(2)} ${currency.toUpperCase()}`;
}

export function formatJobAlertSummary(job: JobAlertPayload): string {
  return `${job.title} (${formatReward(job)} from ${formatJobSource(job.source)})`;
}

export function playJobAlertSound(): boolean {
  if (typeof window === "undefined") {
    return false;
  }

  const audioContext = getOrCreateAudioContext();

  if (!audioContext) {
    return playFallbackAlertSound();
  }

  if (audioContext.state === "suspended") {
    void audioContext
      .resume()
      .then(() => playWebAudioAlertSound(audioContext))
      .catch(() => undefined);
    return playFallbackAlertSound();
  }

  return playWebAudioAlertSound(audioContext) || playFallbackAlertSound();
}

export function unlockJobAlertSound(): boolean {
  const audioContext = getOrCreateAudioContext();
  let attempted = false;

  if (audioContext) {
    attempted = true;
    const prime = () => primeAudioContext(audioContext);
    if (audioContext.state === "suspended") {
      void audioContext
        .resume()
        .then(prime)
        .catch(() => undefined);
    } else {
      prime();
    }
  }

  const fallbackAudio = getFallbackAudioElement();
  if (fallbackAudio) {
    attempted = true;
    try {
      fallbackAudio.muted = true;
      fallbackAudio.currentTime = 0;
      const playPromise = fallbackAudio.play();
      if (playPromise && typeof playPromise.then === "function") {
        void playPromise
          .then(() => {
            fallbackAudio.pause();
            fallbackAudio.currentTime = 0;
            fallbackAudio.muted = false;
          })
          .catch(() => {
            fallbackAudio.muted = false;
          });
      } else {
        fallbackAudio.pause();
        fallbackAudio.currentTime = 0;
        fallbackAudio.muted = false;
      }
    } catch {
      fallbackAudio.muted = false;
    }
  }

  return attempted;
}

function playWebAudioAlertSound(audioContext: AudioContext): boolean {
  try {
    const now = audioContext.currentTime;
    const oscillator = audioContext.createOscillator();
    const gain = audioContext.createGain();

    oscillator.type = "triangle";
    oscillator.frequency.setValueAtTime(880, now);
    oscillator.frequency.exponentialRampToValueAtTime(1320, now + 0.08);
    gain.gain.setValueAtTime(0.0001, now);
    gain.gain.exponentialRampToValueAtTime(0.16, now + 0.015);
    gain.gain.exponentialRampToValueAtTime(0.0001, now + 0.22);

    oscillator.connect(gain);
    gain.connect(audioContext.destination);
    oscillator.start(now);
    oscillator.stop(now + 0.24);
    return true;
  } catch {
    return false;
  }
}

function primeAudioContext(audioContext: AudioContext): void {
  try {
    const now = audioContext.currentTime;
    const oscillator = audioContext.createOscillator();
    const gain = audioContext.createGain();

    oscillator.frequency.setValueAtTime(440, now);
    gain.gain.setValueAtTime(0.00001, now);

    oscillator.connect(gain);
    gain.connect(audioContext.destination);
    oscillator.start(now);
    oscillator.stop(now + 0.03);
  } catch {
    // The fallback media element may still be usable.
  }
}

function getOrCreateAudioContext(): AudioContext | null {
  if (typeof window === "undefined") {
    return null;
  }

  if (sharedAudioContext && sharedAudioContext.state !== "closed") {
    return sharedAudioContext;
  }

  const AudioContextConstructor =
    window.AudioContext ||
    (window as Window & { webkitAudioContext?: typeof AudioContext })
      .webkitAudioContext;

  if (!AudioContextConstructor) {
    return null;
  }

  try {
    sharedAudioContext = new AudioContextConstructor();
    return sharedAudioContext;
  } catch {
    return null;
  }
}

function getFallbackAudioElement(): HTMLAudioElement | null {
  if (typeof window === "undefined" || typeof Audio === "undefined") {
    return null;
  }

  if (fallbackAudioElement) {
    return fallbackAudioElement;
  }

  try {
    fallbackAudioElement = new Audio(getAlertToneDataUrl());
    fallbackAudioElement.preload = "auto";
    fallbackAudioElement.volume = 0.85;
    return fallbackAudioElement;
  } catch {
    return null;
  }
}

function playFallbackAlertSound(): boolean {
  const fallbackAudio = getFallbackAudioElement();
  if (!fallbackAudio) {
    return false;
  }

  try {
    fallbackAudio.muted = false;
    fallbackAudio.currentTime = 0;
    const playPromise = fallbackAudio.play();
    if (playPromise && typeof playPromise.catch === "function") {
      void playPromise.catch(() => undefined);
    }
    return true;
  } catch {
    return false;
  }
}

function getAlertToneDataUrl(): string {
  if (alertToneDataUrl) {
    return alertToneDataUrl;
  }

  const sampleRate = 8000;
  const durationSeconds = 0.28;
  const sampleCount = Math.floor(sampleRate * durationSeconds);
  const bytes = new Uint8Array(44 + sampleCount * 2);
  const view = new DataView(bytes.buffer);

  writeAscii(bytes, 0, "RIFF");
  view.setUint32(4, 36 + sampleCount * 2, true);
  writeAscii(bytes, 8, "WAVE");
  writeAscii(bytes, 12, "fmt ");
  view.setUint32(16, 16, true);
  view.setUint16(20, 1, true);
  view.setUint16(22, 1, true);
  view.setUint32(24, sampleRate, true);
  view.setUint32(28, sampleRate * 2, true);
  view.setUint16(32, 2, true);
  view.setUint16(34, 16, true);
  writeAscii(bytes, 36, "data");
  view.setUint32(40, sampleCount * 2, true);

  for (let i = 0; i < sampleCount; i += 1) {
    const t = i / sampleRate;
    const attack = Math.min(1, t / 0.02);
    const release = Math.max(0, 1 - Math.max(0, t - 0.18) / 0.1);
    const frequency = t < 0.12 ? 880 : 1320;
    const sample =
      Math.sin(2 * Math.PI * frequency * t) * 0.35 * attack * release;
    view.setInt16(44 + i * 2, Math.round(sample * 32767), true);
  }

  let binary = "";
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  alertToneDataUrl = `data:audio/wav;base64,${window.btoa(binary)}`;
  return alertToneDataUrl;
}

function writeAscii(bytes: Uint8Array, offset: number, value: string): void {
  for (let i = 0; i < value.length; i += 1) {
    bytes[offset + i] = value.charCodeAt(i);
  }
}

export function showJobNotification(
  job: JobAlertPayload,
  safeUrl: string,
  bodySuffix?: string,
): boolean {
  if (!canUseBrowserNotifications() || Notification.permission !== "granted") {
    return false;
  }

  const body = `${formatReward(job)} from ${formatJobSource(job.source)}${bodySuffix ? ` - ${bodySuffix}` : ""}`;
  let notification: Notification;
  try {
    notification = new Notification(job.title, {
      body,
      tag: `gengowatcher-job-${job.id}`,
      data: { url: safeUrl },
    });
  } catch {
    return false;
  }

  notification.onclick = () => {
    const opened = openJobWindow(safeUrl);
    if (opened) {
      focusWindow(opened);
    }
    notification.close();
  };

  return true;
}

export function alertForDetectedJob(job: JobAlertPayload): JobAlertResult {
  const safeUrl = getSafeJobUrl(job.url);
  if (!safeUrl) {
    return {
      opened: false,
      notified: false,
      blocked: true,
      reason: "Job URL is not a safe public HTTP(S) URL.",
    };
  }

  let openedWindow: Window | null = null;
  if (preparedJobWindow && !preparedJobWindow.closed) {
    openedWindow = navigateWindow(preparedJobWindow, safeUrl);
    preparedJobWindow = openedWindow;
  } else {
    openedWindow = openJobWindow(safeUrl);
    preparedJobWindow = openedWindow;
  }

  lastKnownJobWindowState = {
    browser_process_alive: Boolean(openedWindow),
    current_url: openedWindow ? safeUrl : undefined,
    current_title: job.title,
    current_action_step: openedWindow
      ? "Opened job page from dashboard"
      : "Job page auto-open blocked",
    current_job_id: job.id,
    logged_in_state: "unknown",
    devtools_connected: false,
  };

  if (openedWindow) {
    focusWindow(openedWindow);
  }

  const opened = Boolean(openedWindow);
  const notified = showJobNotification(
    job,
    safeUrl,
    opened ? "opened automatically" : "popup blocked; click to open",
  );

  return {
    opened,
    notified,
    blocked: !opened,
    reason: opened ? undefined : "Browser blocked the automatic job tab.",
    safeUrl,
  };
}
