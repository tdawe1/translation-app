/**
 * Dashboard Flow Tests
 *
 * Tests for dashboard page behavior including:
 * - Loading and displaying watcher state
 * - Starting and stopping the watcher
 * - Opening and closing the config modal
 * - WebSocket connection indicator
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';
import DashboardPage from '@/app/dashboard/page';
import { ErrorBoundary } from '@/components/error-boundary';

// Mock the modules
vi.mock('@/store/auth', () => ({
  useAuthStore: vi.fn(),
}));

vi.mock('@/store/watcher', () => ({
  useWatcherStore: vi.fn(),
}));

vi.mock('@/hooks/use-watcher-websocket', () => ({
  useWatcherWebSocket: vi.fn(),
}));

// Helper to create a Zustand mock that handles selectors
const createZustandMock = <T,>(state: T) => {
  return (selector?: (state: T) => unknown) => {
    return selector ? selector(state) : state;
  };
};

vi.mock('@/components/auth/protected-route', () => ({
  ProtectedRoute: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

vi.mock('@/components/ui/modal', () => ({
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

vi.mock('@/components/watcher/config-form', () => ({
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

vi.mock('@/components/watcher/job-list', () => ({
  JobList: () => <div data-testid="job-list">Job List Component</div>,
}));

vi.mock('@/lib/api', () => ({
  authApi: {
    logout: vi.fn(),
  },
}));

vi.mock('@/store/toast', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

import { useAuthStore } from '@/store/auth';
import { useWatcherStore } from '@/store/watcher';
import { useWatcherWebSocket } from '@/hooks/use-watcher-websocket';
import { authApi } from '@/lib/api';
import { toast } from '@/store/toast';

const createWebSocketMetrics = (overrides: Partial<ReturnType<typeof useWatcherWebSocket>> = {}) => ({
  connected: true,
  reconnecting: false,
  reconnectCount: 0,
  connectionStartTime: null,
  uptime: 0,
  lastMessageTime: null,
  messagesReceived: 0,
  ...overrides,
});

// Mock window.location
const mockLocation = { href: '' };
Object.defineProperty(window, 'location', {
  value: mockLocation,
  writable: true,
});

describe('Dashboard Flow', () => {
  const mockFetchConfig = vi.fn();
  const mockFetchState = vi.fn();
  const mockStartWatcher = vi.fn();
  const mockStopWatcher = vi.fn();

  const defaultUser = {
    id: 'user-123',
    email: 'test@example.com',
    created_at: '2025-01-01T00:00:00Z',
  };

  const defaultConfig = {
    rss_feed_url: 'https://example.com/feed',
    min_reward: 5.0,
    max_reward: 50.0,
    websocket_enabled: true,
    auto_accept_enabled: false,
  };

  const defaultState = {
    watcher_status: 'stopped' as const,
    total_jobs_found: 42,
    total_earnings: 150.50,
    last_activity: '2025-01-01T12:00:00Z',
  };

  beforeEach(() => {
    vi.clearAllMocks();

    // Default auth state
    const defaultAuthState = {
      user: defaultUser,
      setUser: vi.fn(),
      setLoading: vi.fn(),
      setError: vi.fn(),
      clear: vi.fn(),
      clearToken: vi.fn(),
      fetchUser: vi.fn(),
      isAuthenticated: true,
      isLoading: false,
      error: null,
    };

    // Default watcher state
    const defaultWatcherState = {
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
    };

    // Default mock implementations with selector support
    vi.mocked(useAuthStore).mockImplementation(createZustandMock(defaultAuthState) as any);

    vi.mocked(useWatcherStore).mockImplementation(createZustandMock(defaultWatcherState) as any);

    vi.mocked(useWatcherWebSocket).mockReturnValue(createWebSocketMetrics() as any);

    vi.mocked(authApi.logout).mockResolvedValue(undefined);
    vi.mocked(toast.success).mockImplementation(() => {});
    vi.mocked(toast.error).mockImplementation(() => {});
  });

  describe('loads and displays watcher state', () => {
    it('renders the dashboard with user info', async () => {
      render(<DashboardPage />);

      expect(screen.getByText('GengoWatcher')).toBeInTheDocument();
      expect(screen.getByText('test@example.com')).toBeInTheDocument();
    });

    it('displays watcher status when loaded', async () => {
      render(<DashboardPage />);

      await waitFor(() => {
        expect(screen.getByText('Stopped')).toBeInTheDocument();
      });
    });

    it('displays jobs found and earnings', async () => {
      render(<DashboardPage />);

      await waitFor(() => {
        expect(screen.getByText('42')).toBeInTheDocument(); // Jobs Found
        expect(screen.getByText('$150.50')).toBeInTheDocument(); // Earnings
      });
    });

    it('shows loading state while fetching', async () => {
      vi.mocked(useWatcherStore).mockImplementation(createZustandMock({
        config: null,
        state: null,
        configLoading: true,
        stateLoading: true,
        configError: null,
        stateError: null,
        fetchConfig: mockFetchConfig,
        fetchState: mockFetchState,
        updateConfig: vi.fn(),
        startWatcher: mockStartWatcher,
        stopWatcher: mockStopWatcher,
        setState: vi.fn(),
        clear: vi.fn(),
      }) as any);

      render(<DashboardPage />);

      // Look for the specific loading message text, not just "Loading..."
      expect(screen.getByText('Loading configuration...')).toBeInTheDocument();
    });

    it('displays configuration details', async () => {
      render(<DashboardPage />);

      await waitFor(() => {
        expect(screen.getByText('RSS Feed')).toBeInTheDocument();
        expect(screen.getByText('Min Reward')).toBeInTheDocument();
        expect(screen.getByText('$5.00')).toBeInTheDocument();
        expect(screen.getByText('Enabled')).toBeInTheDocument(); // WebSocket
      });
    });
  });

  describe('starts and stops watcher', () => {
    it('calls startWatcher when Start button is clicked', async () => {
      const user = userEvent.setup();
      vi.mocked(useWatcherStore).mockImplementation(createZustandMock({
        config: defaultConfig,
        state: { ...defaultState, watcher_status: 'stopped' as const },
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
      }) as any);

      render(<DashboardPage />);

      const startButton = screen.getByRole('button', { name: /start watcher/i });
      await user.click(startButton);

      expect(mockStartWatcher).toHaveBeenCalledOnce();
      expect(toast.success).toHaveBeenCalledWith('Watcher started successfully');
    });

    it('calls stopWatcher when Stop button is clicked', async () => {
      const user = userEvent.setup();
      vi.mocked(useWatcherStore).mockImplementation(createZustandMock({
        config: defaultConfig,
        state: { ...defaultState, watcher_status: 'running' as const },
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
      }) as any);

      render(<DashboardPage />);

      const stopButton = screen.getByRole('button', { name: /stop watcher/i });
      await user.click(stopButton);

      expect(mockStopWatcher).toHaveBeenCalledOnce();
      expect(toast.success).toHaveBeenCalledWith('Watcher stopped successfully');
    });

    it('disables Start button when watcher is running', async () => {
      vi.mocked(useWatcherStore).mockImplementation(createZustandMock({
        config: defaultConfig,
        state: { ...defaultState, watcher_status: 'running' as const },
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
      }) as any);

      render(<DashboardPage />);

      const startButton = screen.getByRole('button', { name: /start watcher/i });
      expect(startButton).toBeDisabled();
    });

    it('disables Stop button when watcher is stopped', async () => {
      vi.mocked(useWatcherStore).mockImplementation(createZustandMock({
        config: defaultConfig,
        state: { ...defaultState, watcher_status: 'stopped' as const },
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
      }) as any);

      render(<DashboardPage />);

      const stopButton = screen.getByRole('button', { name: /stop watcher/i });
      expect(stopButton).toBeDisabled();
    });
  });

  describe('opens and closes config modal', () => {
    it('opens modal when Configure button is clicked', async () => {
      const user = userEvent.setup();
      render(<DashboardPage />);

      const configButton = screen.getByRole('button', { name: /configure/i });
      await user.click(configButton);

      expect(screen.getByTestId('modal')).toBeInTheDocument();
      expect(screen.getByTestId('config-form')).toBeInTheDocument();
    });

    it('closes modal when Close button is clicked', async () => {
      const user = userEvent.setup();
      render(<DashboardPage />);

      // Open modal
      const configButton = screen.getByRole('button', { name: /configure/i });
      await user.click(configButton);
      expect(screen.getByTestId('modal')).toBeInTheDocument();

      // Close modal
      const closeButton = screen.getByRole('button', { name: /close/i });
      await user.click(closeButton);

      expect(screen.queryByTestId('modal')).not.toBeInTheDocument();
    });
  });

  describe('WebSocket connection indicator', () => {
    it('shows green dot when connected', async () => {
      vi.mocked(useWatcherWebSocket).mockReturnValue(createWebSocketMetrics({ connected: true }) as any);

      render(<DashboardPage />);

      // The pulsing dot should be in the document when connected
      const pulsingDot = document.querySelector('.animate-pulse');
      expect(pulsingDot).toBeInTheDocument();
    });

    it('hides green dot when not connected', async () => {
      vi.mocked(useWatcherWebSocket).mockReturnValue(createWebSocketMetrics({ connected: false }) as any);

      render(<DashboardPage />);

      // The pulsing dot should not be in the document when not connected
      const pulsingDot = document.querySelector('.animate-pulse');
      expect(pulsingDot).toBeNull();
    });
  });

  describe('logout functionality', () => {
    it('calls logout and redirects when Sign Out is clicked', async () => {
      const user = userEvent.setup();
      render(<DashboardPage />);

      const signOutButton = screen.getByRole('button', { name: /sign out/i });
      await user.click(signOutButton);

      expect(authApi.logout).toHaveBeenCalledOnce();
      expect(mockLocation.href).toBe('/');
    });
  });

  describe('error handling', () => {
    it('shows error message when startWatcher fails', async () => {
      const user = userEvent.setup();
      const error = new Error('Failed to start watcher');
      mockStartWatcher.mockRejectedValueOnce(error);

      vi.mocked(useWatcherStore).mockImplementation(createZustandMock({
        config: defaultConfig,
        state: { ...defaultState, watcher_status: 'stopped' as const },
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
      }) as any);

      render(<DashboardPage />);

      const startButton = screen.getByRole('button', { name: /start watcher/i });
      await user.click(startButton);

      expect(toast.error).toHaveBeenCalledWith('Failed to start watcher');
    });

    it('shows error message when stopWatcher fails', async () => {
      const user = userEvent.setup();
      const error = new Error('Failed to stop watcher');
      mockStopWatcher.mockRejectedValueOnce(error);

      vi.mocked(useWatcherStore).mockImplementation(createZustandMock({
        config: defaultConfig,
        state: { ...defaultState, watcher_status: 'running' as const },
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
      }) as any);

      render(<DashboardPage />);

      const stopButton = screen.getByRole('button', { name: /stop watcher/i });
      await user.click(stopButton);

      expect(toast.error).toHaveBeenCalledWith('Failed to stop watcher');
    });

    it('displays config error when present', async () => {
      vi.mocked(useWatcherStore).mockImplementation(createZustandMock({
        config: null,
        state: defaultState,
        configLoading: false,
        stateLoading: false,
        configError: 'Failed to load configuration',
        stateError: null,
        fetchConfig: mockFetchConfig,
        fetchState: mockFetchState,
        updateConfig: vi.fn(),
        startWatcher: mockStartWatcher,
        stopWatcher: mockStopWatcher,
        setState: vi.fn(),
        clear: vi.fn(),
      }) as any);

      render(<DashboardPage />);

      expect(screen.getByText('Failed to load configuration')).toBeInTheDocument();
    });
  });

  describe('Error Boundary', () => {
    // Component that throws an error on button click
    function ThrowOnClickComponent() {
      const [shouldThrow, setShouldThrow] = React.useState(false);

      if (shouldThrow) {
        throw new Error('Test error from component');
      }

      return (
        <div>
          <div data-testid="safe-content">Safe content</div>
          <button onClick={() => setShouldThrow(true)}>Trigger Error</button>
        </div>
      );
    }

    it('catches errors and displays fallback UI', () => {
      const onError = vi.fn();
      const { getByTestId, getByRole } = render(
        <ErrorBoundary onError={onError}>
          <ThrowOnClickComponent />
        </ErrorBoundary>,
      );

      expect(getByTestId('safe-content')).toBeInTheDocument();

      // Click button to trigger error
      act(() => {
        getByRole('button').click();
      });

      // ErrorBoundary catches the error and shows fallback
      expect(screen.getByText('Something went wrong')).toBeInTheDocument();
      expect(onError).toHaveBeenCalled();
    });

    it('displays custom fallback when provided', () => {
      const customFallback = <div data-testid="custom-fallback">Custom Error UI</div>;

      const { getByRole } = render(
        <ErrorBoundary fallback={customFallback}>
          <ThrowOnClickComponent />
        </ErrorBoundary>,
      );

      // Click button to trigger error
      act(() => {
        getByRole('button').click();
      });

      expect(screen.getByTestId('custom-fallback')).toBeInTheDocument();
    });

    it('displays error message in fallback UI', () => {
      const onError = vi.fn();

      const { getByRole } = render(
        <ErrorBoundary onError={onError}>
          <ThrowOnClickComponent />
        </ErrorBoundary>,
      );

      // Click button to trigger error
      act(() => {
        getByRole('button').click();
      });

      expect(screen.getByText('Test error from component')).toBeInTheDocument();
    });
  });
});
