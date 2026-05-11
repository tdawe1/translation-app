import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  alertForDetectedJob,
  closePreparedJobAlertWindow,
  getPreparedJobAlertWindowState,
  getSafeJobUrl,
  playJobAlertSound,
  prepareJobAlertWindow,
  unlockJobAlertSound,
} from "./job-alerts";

const job = {
  id: "job-1",
  title: "Japanese to English",
  reward: 12.5,
  url: "https://gengo.com/dashboard/jobs/job-1",
  source: "external",
};

describe("job alerts", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
    closePreparedJobAlertWindow();
  });

  it("blocks non-public or non-http job URLs", () => {
    expect(getSafeJobUrl("http://127.0.0.1:8000/job")).toBeNull();
    expect(getSafeJobUrl("http://192.168.1.5/job")).toBeNull();
    expect(getSafeJobUrl("file:///tmp/job")).toBeNull();
  });

  it("opens safe job URLs automatically", () => {
    const openedWindow = {
      opener: {},
      focus: vi.fn(),
      closed: false,
      location: {
        href: "",
      },
    } as unknown as Window;
    const open = vi.spyOn(window, "open").mockReturnValue(openedWindow);

    const result = alertForDetectedJob(job);

    expect(open).toHaveBeenCalledWith("about:blank", "gengowatcher-job-window");
    expect(result).toMatchObject({
      opened: true,
      blocked: false,
      safeUrl: job.url,
    });
    expect(openedWindow.opener).toBeNull();
    expect(openedWindow.location.href).toBe(job.url);
    expect(getPreparedJobAlertWindowState()).toMatchObject({
      browser_process_alive: true,
      current_url: job.url,
      current_title: job.title,
      current_job_id: job.id,
    });
  });

  it("retargets an already cross-origin prepared job window by name", () => {
    const preparedLocation = {};
    Object.defineProperty(preparedLocation, "href", {
      set() {
        throw new Error(
          'Permission denied to access property "href" on cross-origin object',
        );
      },
    });
    Object.defineProperty(preparedLocation, "assign", {
      get() {
        throw new Error(
          "location.assign should not be read for cross-origin windows",
        );
      },
    });

    const preparedWindow = {
      opener: {},
      focus: vi.fn(),
      close: vi.fn(),
      closed: false,
      document: {
        title: "",
        body: {
          innerHTML: "",
        },
      },
      location: preparedLocation,
    } as unknown as Window;
    const namedWindow = {
      opener: null,
      focus: vi.fn(),
      closed: false,
      location: {
        href: "",
      },
    } as unknown as Window;
    const open = vi
      .spyOn(window, "open")
      .mockReturnValueOnce(preparedWindow)
      .mockReturnValueOnce(namedWindow);

    expect(prepareJobAlertWindow()).toBe(true);
    const result = alertForDetectedJob(job);

    expect(open).toHaveBeenNthCalledWith(
      1,
      "about:blank",
      "gengowatcher-job-window",
    );
    expect(open).toHaveBeenNthCalledWith(2, job.url, "gengowatcher-job-window");
    expect(result).toMatchObject({
      opened: true,
      blocked: false,
    });
    expect(namedWindow.focus).toHaveBeenCalled();
  });

  it("falls back to a browser notification when auto-open is blocked", () => {
    const notifications: Array<{
      title: string;
      options?: NotificationOptions;
      onclick: (() => void) | null;
      close: () => void;
    }> = [];

    class MockNotification {
      static permission: NotificationPermission = "granted";

      onclick: (() => void) | null = null;
      close = vi.fn();

      constructor(
        public title: string,
        public options?: NotificationOptions,
      ) {
        notifications.push(this);
      }
    }

    vi.stubGlobal("Notification", MockNotification);
    const open = vi.spyOn(window, "open").mockReturnValue(null);

    const result = alertForDetectedJob(job);

    expect(result).toMatchObject({
      opened: false,
      notified: true,
      blocked: true,
    });
    expect(notifications).toHaveLength(1);
    expect(notifications[0].title).toBe(job.title);

    notifications[0].onclick?.();
    expect(open).toHaveBeenCalledTimes(2);
  });

  it("primes and plays fallback audio when Web Audio is unavailable", async () => {
    const play = vi.fn(() => Promise.resolve());
    const pause = vi.fn();

    class MockAudio {
      currentTime = 0;
      muted = false;
      preload = "";
      src: string;
      volume = 1;

      constructor(src: string) {
        this.src = src;
      }

      play = play;
      pause = pause;
    }

    vi.stubGlobal("AudioContext", undefined);
    vi.stubGlobal("webkitAudioContext", undefined);
    vi.stubGlobal("Audio", MockAudio);

    expect(unlockJobAlertSound()).toBe(true);
    await Promise.resolve();

    expect(play).toHaveBeenCalledTimes(1);
    expect(pause).toHaveBeenCalledTimes(1);

    expect(playJobAlertSound()).toBe(true);
    expect(play).toHaveBeenCalledTimes(2);
  });
});
