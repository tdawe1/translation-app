import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AuthForm, OAuthButtons } from './AuthForm';

describe('AuthForm', () => {
  const mockOnSubmit = vi.fn();

  beforeEach(() => {
    mockOnSubmit.mockClear();
  });

  describe('login mode', () => {
    it('should render email and password fields', () => {
      render(<AuthForm mode="login" onSubmit={mockOnSubmit} />);

      expect(screen.getByLabelText(/email/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/password/i)).toBeInTheDocument();
    });

    it('should not show confirm password field in login mode', () => {
      render(<AuthForm mode="login" onSubmit={mockOnSubmit} />);

      expect(screen.queryByLabelText(/confirm password/i)).not.toBeInTheDocument();
    });

    it('should submit email and password', async () => {
      const user = userEvent.setup();
      mockOnSubmit.mockResolvedValue(undefined);
      render(<AuthForm mode="login" onSubmit={mockOnSubmit} />);

      await user.type(screen.getByLabelText(/email/i), 'test@example.com');
      await user.type(screen.getByLabelText(/password/i), 'password123');
      await user.click(screen.getByRole('button', { name: /sign in/i }));

      await waitFor(() => {
        expect(mockOnSubmit).toHaveBeenCalledWith({
          email: 'test@example.com',
          password: 'password123',
        });
      });
    });

    it('should not render a magic link link inside the shared form', () => {
      render(<AuthForm mode="login" onSubmit={mockOnSubmit} />);

      expect(
        screen.queryByRole('link', { name: /send magic link instead/i }),
      ).not.toBeInTheDocument();
    });
  });

  describe('register mode', () => {
    it('should show confirm password field', () => {
      render(<AuthForm mode="register" onSubmit={mockOnSubmit} />);

      expect(screen.getByLabelText(/confirm password/i)).toBeInTheDocument();
    });

    it('should validate password match', async () => {
      const user = userEvent.setup();
      render(<AuthForm mode="register" onSubmit={mockOnSubmit} />);

      // Fill in email (required field) and passwords
      await user.type(screen.getByLabelText(/email/i), 'test@example.com');
      const passwordInputs = screen.getAllByLabelText(/password/i);
      await user.type(passwordInputs[0], 'password123');
      await user.type(passwordInputs[1], 'different');

      // Submit the form by clicking the submit button
      await user.click(screen.getByRole('button', { name: /create account/i }));

      // Check for error message - use queryBy with waitFor
      const error = await screen.findByText(/passwords do not match/i);
      expect(error).toBeInTheDocument();
      expect(mockOnSubmit).not.toHaveBeenCalled();
    });

    it('should validate password length', async () => {
      const user = userEvent.setup();
      render(<AuthForm mode="register" onSubmit={mockOnSubmit} />);

      // Fill in email (required field) and passwords
      await user.type(screen.getByLabelText(/email/i), 'test@example.com');
      const passwordInputs = screen.getAllByLabelText(/password/i);
      await user.type(passwordInputs[0], 'short');
      await user.type(passwordInputs[1], 'short');

      // Submit the form
      await user.click(screen.getByRole('button', { name: /create account/i }));

      // Check for error message - use queryBy with waitFor
      const error = await screen.findByText(/at least 8 characters/i);
      expect(error).toBeInTheDocument();
      expect(mockOnSubmit).not.toHaveBeenCalled();
    });

    it('should submit with all fields when valid', async () => {
      const user = userEvent.setup();
      mockOnSubmit.mockResolvedValue(undefined);
      render(<AuthForm mode="register" onSubmit={mockOnSubmit} />);

      await user.type(screen.getByLabelText(/email/i), 'test@example.com');
      // Use more specific query - the password field comes first
      const passwordInputs = screen.getAllByLabelText(/password/i);
      await user.type(passwordInputs[0], 'password123');
      await user.type(passwordInputs[1], 'password123');
      await user.click(screen.getByRole('button', { name: /create account/i }));

      await waitFor(() => {
        expect(mockOnSubmit).toHaveBeenCalledWith({
          email: 'test@example.com',
          password: 'password123',
          confirmPassword: 'password123',
        });
      });
    });

    it('should not show magic link link in register mode', () => {
      render(<AuthForm mode="register" onSubmit={mockOnSubmit} />);

      expect(screen.queryByRole('link', { name: /send magic link instead/i })).not.toBeInTheDocument();
    });
  });

  describe('loading state', () => {
    it('should disable inputs and show loading text when isLoading is true', () => {
      render(<AuthForm mode="login" onSubmit={mockOnSubmit} isLoading={true} />);

      expect(screen.getByRole('button', { name: /working/i })).toBeInTheDocument();
      expect(screen.getByLabelText(/email/i)).toBeDisabled();
      expect(screen.getByLabelText(/password/i)).toBeDisabled();
    });

    it('should show shared loading text for register mode', () => {
      render(<AuthForm mode="register" onSubmit={mockOnSubmit} isLoading={true} />);

      expect(screen.getByRole('button', { name: /working/i })).toBeInTheDocument();
    });
  });

  describe('error handling', () => {
    it('should display error message from prop', () => {
      render(
        <AuthForm
          mode="login"
          onSubmit={mockOnSubmit}
          errorMessage="Invalid credentials"
        />
      );

      expect(screen.getByText(/invalid credentials/i)).toBeInTheDocument();
    });

    it('should display error message from onSubmit rejection', async () => {
      const user = userEvent.setup();
      mockOnSubmit.mockRejectedValue(new Error('Network error'));
      render(<AuthForm mode="login" onSubmit={mockOnSubmit} />);

      await user.type(screen.getByLabelText(/email/i), 'test@example.com');
      await user.type(screen.getByLabelText(/password/i), 'password123');
      await user.click(screen.getByRole('button', { name: /sign in/i }));

      await waitFor(() => {
        expect(screen.getByText(/network error/i)).toBeInTheDocument();
      });
    });

    it('should clear error when user types', async () => {
      const user = userEvent.setup();
      render(
        <AuthForm
          mode="login"
          onSubmit={mockOnSubmit}
          errorMessage="Previous error"
        />
      );

      const error = screen.getByText(/previous error/i);
      expect(error).toBeInTheDocument();

      await user.type(screen.getByLabelText(/email/i), 'a');

      expect(error).not.toBeInTheDocument();
    });
  });
});

describe('OAuthButtons', () => {
  const mockOnOAuthLogin = vi.fn();

  beforeEach(() => {
    mockOnOAuthLogin.mockClear();
  });

  it('should render Google and GitHub buttons', () => {
    render(<OAuthButtons onOAuthLogin={mockOnOAuthLogin} />);

    expect(screen.getByRole('button', { name: /continue with google/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /continue with github/i })).toBeInTheDocument();
  });

  it('should call onOAuthLogin with google provider', async () => {
    const user = userEvent.setup();
    mockOnOAuthLogin.mockResolvedValue(undefined);
    render(<OAuthButtons onOAuthLogin={mockOnOAuthLogin} />);

    await user.click(screen.getByRole('button', { name: /continue with google/i }));

    await waitFor(() => {
      expect(mockOnOAuthLogin).toHaveBeenCalledWith('google');
    });
  });

  it('should call onOAuthLogin with github provider', async () => {
    const user = userEvent.setup();
    mockOnOAuthLogin.mockResolvedValue(undefined);
    render(<OAuthButtons onOAuthLogin={mockOnOAuthLogin} />);

    await user.click(screen.getByRole('button', { name: /continue with github/i }));

    await waitFor(() => {
      expect(mockOnOAuthLogin).toHaveBeenCalledWith('github');
    });
  });

  it('should display error when OAuth fails', async () => {
    const user = userEvent.setup();
    mockOnOAuthLogin.mockRejectedValue(new Error('OAuth failed'));
    render(<OAuthButtons onOAuthLogin={mockOnOAuthLogin} />);

    await user.click(screen.getByRole('button', { name: /continue with google/i }));

    await waitFor(() => {
      expect(screen.getByText(/oauth failed/i)).toBeInTheDocument();
    });
  });

  it('should disable buttons when disabled prop is true', () => {
    render(<OAuthButtons onOAuthLogin={mockOnOAuthLogin} disabled={true} />);

    expect(screen.getByRole('button', { name: /continue with google/i })).toBeDisabled();
    expect(screen.getByRole('button', { name: /continue with github/i })).toBeDisabled();
  });
});
