/**
 * Dashboard Flow Tests
 *
 * Tests for dashboard page behavior including:
 * - Loading and displaying watcher state
 * - Starting and stopping the watcher
 * - Opening and closing the config modal
 * - WebSocket connection indicator
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen, waitFor, act } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";
import DashboardPage from "@/app/dashboard/page";
import { ErrorBoundary } from "@/components/error-boundary";

// Mock the modules
vi.mock("@/store/auth", () => ({
  useAuthStore: vi.fn(),
}));

vi.mock("@/store/watcher", () => ({
  useWatcherStore: vi.fn(),
}));

vi.mock("@/hooks/use-watcher-websocket", () => ({
  useWatcherWebSocket: vi.fn(),
}));

// Helper to create a Zustand mock that handles selectors
const createZustandMock = <T,>(state: T) => {
  return ((selector?: (state: T) => unknown) => {
    return selector ? selector(state) : state;
  }) as any;
};

vi.mock("@/components/auth/protected-route", () => ({
  ProtectedRoute: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
}));

vi.mock("@/components/ui/modal", () => ({
  Modal: ({
    isOpen,
    onClose,
    children,
  }: {
    isOpen: boolean;
    onClose: () => void;
    children: React.ReactNode;
  }) => (isOpen ? <div data-testid="modal">{children}</div> : null),
}));

vi.mock("@/components/watcher/config-form", () => ({
  WatcherConfigForm: ({ onClose }: { onClose: () => void }) => (
    <div>
      <button onClick={onClose}>Close</button>
      <form data-testid="config-form">
        <input name="rss_feed_url" placeholder="RSS Feed URL" />
        <input name="min_reward" type="number" placeholder="Min Reward" />
      </form>
    </div>
  ),
}));

vi.mock("@/components/watcher/job-list", () => ({
  JobList: () => <div data-testid="job-list">Job List Component</div>,
}));

vi.mock("@/lib/api", () => ({
  authApi: {
    logout: vi.fn(),
  },
  watcherApi: {
    getEvents: vi.fn(),
    syncBrowserState: vi.fn(),
    restartBrowser: vi.fn(),
    captureBrowserScreenshot: vi.fn(),
  },
}));

vi.mock("@/store/toast", () => ({
  toast: {
    success: vi.fn(),
    info: vi.fn(),
    error: vi.fn(),
  },
}));

import { useAuthStore } from "@/store/auth";
import { useWatcherStore } from "@/store/watcher";
import { useWatcherWebSocket } from "@/hooks/use-watcher-websocket";
import { authApi, watcherApi } from "@/lib/api";
import { toast } from "@/store/toast";
import { useRealtimeStore } from "@/store/realtime";

// Mock window.location
const mockLocation = { href: "" };
Object.defineProperty(window, "location", {
  value: mockLocation,
  writable: true,
});

describe("Dashboard Flow", () => {
  const mockFetchConfig = vi.fn();
  const mockFetchState = vi.fn();
  const mockStartWatcher = vi.fn();
  const mockStopWatcher = vi.fn();

  const defaultUser = {
    id: "user-123",
    email: "test@example.com",
    created_at: "2025-01-01T00:00:00Z",
  };

  const defaultConfig = {
    rss_feed_url: "https://example.com/feed",
    min_reward: 5.0,
    max_reward: 50.0,
    websocket_enabled: true,
    auto_accept_enabled: false,
  };

  const defaultState = {
    watcher_status: "stopped" as const,
    overall_status: "stopped",
    feed_status: "stopped",
    browser_status: "unconfigured",
    action_status: "idle",
    alert_status: "none",
    profile_status: "unseeded",
    current_action_step: "",
    current_job_id: "",
    last_error: "",
    worker_id: "user-123",
    total_jobs_found: 42,
    total_earnings: 150.5,
    last_activity: "2025-01-01T12:00:00Z",
  };

  const createWatcherStoreState = (overrides = {}) => ({
    config: defaultConfig,
    state: defaultState,
    configLoading: false,
    stateLoading: false,
    configError: null,
    stateError: null,
    fetchConfig: mockFetchConfig,
    fetchState: mockFetchState,
    updateConfig: vi.fn(),
    startWatcher: mockStartWatcher,
    stopWatcher: mockStopWatcher,
    setState: vi.fn(),
    clear: vi.fn(),
    ...overrides,
  });

  const createWebSocketState = (overrides = {}) => ({
    connected: true,
    reconnecting: false,
    reconnectCount: 0,
    connectionStartTime: null,
    uptime: 0,
    lastMessageTime: null,
    messagesReceived: 0,
    connect: vi.fn(),
    disconnect: vi.fn(),
    ...overrides,
  });

  beforeEach(() => {
    vi.clearAllMocks();
    vi.useRealTimers();
    useRealtimeStore.getState().clearEvents();
    useRealtimeStore.getState().resetStats();

    // Default auth state
    const defaultAuthState = {
      user: defaultUser,
      setUser: vi.fn(),
      setLoading: vi.fn(),
      setError: vi.fn(),
      clear: vi.fn(),
      isAuthenticated: true,
      isLoading: false,
      error: null,
    };

    // Default watcher state
    // Default mock implementations with selector support
    vi.mocked(useAuthStore).mockImplementation(
      createZustandMock(defaultAuthState),
    );

    vi.mocked(useWatcherStore).mockImplementation(
      createZustandMock(createWatcherStoreState()),
    );

    vi.mocked(useWatcherWebSocket).mockReturnValue(createWebSocketState());

    vi.mocked(authApi.logout).mockResolvedValue(undefined);
    vi.mocked(watcherApi.getEvents).mockResolvedValue({ events: [] });
    vi.mocked(watcherApi.syncBrowserState).mockResolvedValue({
      status: "synced",
    });
    vi.mocked(watcherApi.restartBrowser).mockResolvedValue({
      status: "restarted",
    });
    vi.mocked(watcherApi.captureBrowserScreenshot).mockResolvedValue({
      status: "captured",
      screenshot_artifact_id: "manual.png",
    });
    vi.mocked(toast.success).mockImplementation(() => {});
    vi.mocked(toast.info).mockImplementation(() => {});
    vi.mocked(toast.error).mockImplementation(() => {});
  });

  describe("loads and displays watcher state", () => {
    it("renders the dashboard with user info", async () => {
      render(<DashboardPage />);

      expect(screen.getByText("GengoWatcher")).toBeInTheDocument();
      expect(screen.getByText("test@example.com")).toBeInTheDocument();
    });

    it("displays watcher status when loaded", async () => {
      render(<DashboardPage />);

      await waitFor(() => {
        expect(screen.getByText("Stopped")).toBeInTheDocument();
      });
    });

    it("displays jobs found and earnings", async () => {
      render(<DashboardPage />);

      await waitFor(() => {
        expect(screen.getByText("42")).toBeInTheDocument(); // Jobs Found
        expect(screen.getByText("$150.50")).toBeInTheDocument(); // Earnings
      });
    });

    it("shows a live overview clock and recent watcher output", async () => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date("2025-01-01T12:00:00Z"));
      vi.mocked(watcherApi.getEvents).mockResolvedValue({
        events: [
          {
            id: "event-1",
            type: "job.matched",
            level: "info",
            source: "watcher",
            message: "Matched Japanese legal translation job",
            data: {},
            occurred_at: "2025-01-01T12:00:00Z",
          },
        ],
      });
      useRealtimeStore.getState().setEvents([
        {
          id: "event-1",
          type: "job.matched",
          level: "info",
          source: "watcher",
          message: "Matched Japanese legal translation job",
          data: {},
          timestamp: "2025-01-01T12:00:00Z",
        },
      ]);

      render(<DashboardPage />);

      expect(screen.getByTestId("live-clock")).toBeInTheDocument();
      const initialClock = screen.getByTestId("live-clock").textContent;

      expect(
        screen.getByText("Matched Japanese legal translation job"),
      ).toBeInTheDocument();
      expect(screen.getByTestId("overview-terminal-output")).toHaveTextContent(
        "watcher job.matched",
      );

      await act(async () => {
        vi.advanceTimersByTime(1_000);
      });

      expect(screen.getByTestId("live-clock").textContent).not.toBe(
        initialClock,
      );
      vi.useRealTimers();
    });

    it("keeps the overview usable while fetching", async () => {
      vi.mocked(useWatcherStore).mockImplementation(
        createZustandMock(
          createWatcherStoreState({
            config: null,
            state: null,
            configLoading: true,
            stateLoading: true,
          }),
        ),
      );

      render(<DashboardPage />);

      expect(screen.getByTestId("live-clock")).toBeInTheDocument();
      expect(screen.getByText("Loading...")).toBeInTheDocument();
    });

    it("keeps configuration details behind the Configure action", async () => {
      const user = userEvent.setup();
      render(<DashboardPage />);

      expect(screen.queryByText("RSS Feed")).not.toBeInTheDocument();

      await user.click(screen.getByRole("button", { name: /configure/i }));

      expect(screen.getByTestId("config-form")).toBeInTheDocument();
    });

    it("displays runtime domain statuses in the operations section", async () => {
      const user = userEvent.setup();
      vi.mocked(useWatcherStore).mockImplementation(
        createZustandMock(
          createWatcherStoreState({
            state: {
              ...defaultState,
              watcher_status: "running" as const,
              overall_status: "degraded",
              feed_status: "monitoring",
              browser_status: "unconfigured",
              action_status: "idle",
              alert_status: "warning",
            },
          }),
        ),
      );

      render(<DashboardPage />);

      await user.click(
        screen.getAllByRole("button", { name: /^operations$/i })[0],
      );

      expect(await screen.findByText("Overall")).toBeInTheDocument();
      expect(screen.getByText("Feeds")).toBeInTheDocument();
      expect(screen.getByText("Browser")).toBeInTheDocument();
      expect(screen.getByText("Action")).toBeInTheDocument();
      expect(screen.getByText("Alerts")).toBeInTheDocument();
      expect(screen.getByText("degraded")).toBeInTheDocument();
      expect(screen.getByText("monitoring")).toBeInTheDocument();
      expect(screen.getByText("unconfigured")).toBeInTheDocument();
      expect(screen.getByText("idle")).toBeInTheDocument();
      expect(screen.getByText("warning")).toBeInTheDocument();
    });

    it("switches between dashboard sections without leaving the page", async () => {
      const user = userEvent.setup();
      render(<DashboardPage />);

      expect(screen.getByTestId("live-clock")).toBeInTheDocument();
      expect(screen.queryByTestId("job-list")).not.toBeInTheDocument();

      await user.click(screen.getAllByRole("button", { name: /^jobs$/i })[0]);
      expect(screen.getByText("Detected jobs")).toBeInTheDocument();
      expect(screen.getByTestId("job-list")).toBeInTheDocument();

      await user.click(
        screen.getAllByRole("button", { name: /^activity$/i })[0],
      );
      expect(screen.getByText("Realtime activity")).toBeInTheDocument();
      expect(screen.queryByTestId("job-list")).not.toBeInTheDocument();
      expect(screen.getByTestId("realtime-event-log")).toBeInTheDocument();
    });

    it("can collapse the dashboard sidebar", async () => {
      const user = userEvent.setup();
      render(<DashboardPage />);

      expect(screen.getByText("Workspace")).toBeInTheDocument();

      await user.click(
        screen.getByRole("button", { name: /collapse sidebar/i }),
      );

      expect(screen.queryByText("Workspace")).not.toBeInTheDocument();
      expect(
        screen.getByRole("button", { name: /expand sidebar/i }),
      ).toBeInTheDocument();
    });

    it("keeps the dashboard usable when watcher events fail to load", async () => {
      const consoleError = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});
      vi.mocked(watcherApi.getEvents).mockRejectedValue(
        new Error("timeline offline"),
      );

      render(<DashboardPage />);

      expect(await screen.findByText("GengoWatcher")).toBeInTheDocument();

      await waitFor(() => {
        expect(mockFetchConfig).toHaveBeenCalled();
        expect(mockFetchState).toHaveBeenCalled();
        expect(consoleError).toHaveBeenCalledWith(
          "Failed to load watcher events",
          expect.any(Error),
        );
      });

      consoleError.mockRestore();
    });
  });

  describe("starts and stops watcher", () => {
    it("calls startWatcher when Start button is clicked", async () => {
      const user = userEvent.setup();
      vi.mocked(useWatcherStore).mockImplementation(
        createZustandMock(
          createWatcherStoreState({
            state: { ...defaultState, watcher_status: "stopped" as const },
          }),
        ),
      );

      render(<DashboardPage />);

      const startButton = screen.getByRole("button", {
        name: /start watcher/i,
      });
      await user.click(startButton);

      expect(mockStartWatcher).toHaveBeenCalledOnce();
      expect(toast.success).toHaveBeenCalledWith(
        "Watcher started successfully",
      );
    });

    it("calls stopWatcher when Stop button is clicked", async () => {
      const user = userEvent.setup();
      vi.mocked(useWatcherStore).mockImplementation(
        createZustandMock(
          createWatcherStoreState({
            state: { ...defaultState, watcher_status: "running" as const },
          }),
        ),
      );

      render(<DashboardPage />);

      const stopButton = screen.getByRole("button", { name: /stop watcher/i });
      await user.click(stopButton);

      expect(mockStopWatcher).toHaveBeenCalledOnce();
      expect(toast.success).toHaveBeenCalledWith(
        "Watcher stopped successfully",
      );
    });

    it("disables Start button when watcher is running", async () => {
      vi.mocked(useWatcherStore).mockImplementation(
        createZustandMock(
          createWatcherStoreState({
            state: { ...defaultState, watcher_status: "running" as const },
          }),
        ),
      );

      render(<DashboardPage />);

      const startButton = screen.getByRole("button", {
        name: /start watcher/i,
      });
      expect(startButton).toBeDisabled();
    });

    it("disables Stop button when watcher is stopped", async () => {
      vi.mocked(useWatcherStore).mockImplementation(
        createZustandMock(
          createWatcherStoreState({
            state: { ...defaultState, watcher_status: "stopped" as const },
          }),
        ),
      );

      render(<DashboardPage />);

      const stopButton = screen.getByRole("button", { name: /stop watcher/i });
      expect(stopButton).toBeDisabled();
    });
  });

  describe("opens and closes config modal", () => {
    it("opens modal when Configure button is clicked", async () => {
      const user = userEvent.setup();
      render(<DashboardPage />);

      const configButton = screen.getByRole("button", { name: /configure/i });
      await user.click(configButton);

      expect(screen.getByTestId("modal")).toBeInTheDocument();
      expect(screen.getByTestId("config-form")).toBeInTheDocument();
    });

    it("closes modal when Close button is clicked", async () => {
      const user = userEvent.setup();
      render(<DashboardPage />);

      // Open modal
      const configButton = screen.getByRole("button", { name: /configure/i });
      await user.click(configButton);
      expect(screen.getByTestId("modal")).toBeInTheDocument();

      // Close modal
      const closeButton = screen.getByRole("button", { name: /close/i });
      await user.click(closeButton);

      expect(screen.queryByTestId("modal")).not.toBeInTheDocument();
    });
  });

  describe("WebSocket connection indicator", () => {
    it("shows green dot when connected", async () => {
      vi.mocked(useWatcherWebSocket).mockReturnValue(createWebSocketState());

      render(<DashboardPage />);

      // The pulsing dot should be in the document when connected
      const pulsingDot = document.querySelector(".animate-pulse");
      expect(pulsingDot).toBeInTheDocument();
    });

    it("hides green dot when not connected", async () => {
      vi.mocked(useWatcherWebSocket).mockReturnValue(
        createWebSocketState({
          connected: false,
        }),
      );

      render(<DashboardPage />);

      // The pulsing dot should not be in the document when not connected
      const pulsingDot = document.querySelector(".animate-pulse");
      expect(pulsingDot).toBeNull();
    });
  });

  describe("logout functionality", () => {
    it("calls logout and redirects when Sign Out is clicked", async () => {
      const user = userEvent.setup();
      render(<DashboardPage />);

      const signOutButton = screen.getByRole("button", { name: /sign out/i });
      await user.click(signOutButton);

      expect(authApi.logout).toHaveBeenCalledOnce();
      expect(mockLocation.href).toBe("/");
    });
  });

  describe("error handling", () => {
    it("shows error message when startWatcher fails", async () => {
      const user = userEvent.setup();
      const error = new Error("Failed to start watcher");
      mockStartWatcher.mockRejectedValueOnce(error);

      vi.mocked(useWatcherStore).mockImplementation(
        createZustandMock(
          createWatcherStoreState({
            state: { ...defaultState, watcher_status: "stopped" as const },
          }),
        ),
      );

      render(<DashboardPage />);

      const startButton = screen.getByRole("button", {
        name: /start watcher/i,
      });
      await user.click(startButton);

      expect(toast.error).toHaveBeenCalledWith("Failed to start watcher");
    });

    it("shows error message when stopWatcher fails", async () => {
      const user = userEvent.setup();
      const error = new Error("Failed to stop watcher");
      mockStopWatcher.mockRejectedValueOnce(error);

      vi.mocked(useWatcherStore).mockImplementation(
        createZustandMock(
          createWatcherStoreState({
            state: { ...defaultState, watcher_status: "running" as const },
          }),
        ),
      );

      render(<DashboardPage />);

      const stopButton = screen.getByRole("button", { name: /stop watcher/i });
      await user.click(stopButton);

      expect(toast.error).toHaveBeenCalledWith("Failed to stop watcher");
    });

    it("does not show configuration errors on the overview", async () => {
      vi.mocked(useWatcherStore).mockImplementation(
        createZustandMock(
          createWatcherStoreState({
            config: null,
            state: defaultState,
            configError: "Failed to load configuration",
          }),
        ),
      );

      render(<DashboardPage />);

      expect(
        screen.queryByText("Failed to load configuration"),
      ).not.toBeInTheDocument();
    });
  });

  describe("Error Boundary", () => {
    // Component that throws an error on button click
    function ThrowOnClickComponent() {
      const [shouldThrow, setShouldThrow] = React.useState(false);

      if (shouldThrow) {
        throw new Error("Test error from component");
      }

      return (
        <div>
          <div data-testid="safe-content">Safe content</div>
          <button onClick={() => setShouldThrow(true)}>Trigger Error</button>
        </div>
      );
    }

    it("catches errors and displays fallback UI", () => {
      const onError = vi.fn();
      const { getByTestId, getByRole } = render(
        <ErrorBoundary onError={onError}>
          <ThrowOnClickComponent />
        </ErrorBoundary>,
      );

      expect(getByTestId("safe-content")).toBeInTheDocument();

      // Click button to trigger error
      act(() => {
        getByRole("button").click();
      });

      // ErrorBoundary catches the error and shows fallback
      expect(screen.getByText("Something went wrong")).toBeInTheDocument();
      expect(onError).toHaveBeenCalled();
    });

    it("displays custom fallback when provided", () => {
      const customFallback = (
        <div data-testid="custom-fallback">Custom Error UI</div>
      );

      const { getByRole } = render(
        <ErrorBoundary fallback={customFallback}>
          <ThrowOnClickComponent />
        </ErrorBoundary>,
      );

      // Click button to trigger error
      act(() => {
        getByRole("button").click();
      });

      expect(screen.getByTestId("custom-fallback")).toBeInTheDocument();
    });

    it("displays error message in fallback UI", () => {
      const onError = vi.fn();

      const { getByRole } = render(
        <ErrorBoundary onError={onError}>
          <ThrowOnClickComponent />
        </ErrorBoundary>,
      );

      // Click button to trigger error
      act(() => {
        getByRole("button").click();
      });

      expect(screen.getByText("Test error from component")).toBeInTheDocument();
    });
  });
});
