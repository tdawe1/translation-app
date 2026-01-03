// Package oauth provides OAuth2 client functionality
package oauth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

// GoogleUserInfo represents the response from Google's userinfo endpoint
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
}

// GitHubUserEmail represents the email response from GitHub (may be null)
type GitHubUserEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// GitHubUserInfo represents the response from GitHub's user endpoint
type GitHubUserInfo struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// FetchGoogleUserInfo fetches user info from Google using the access token
func FetchGoogleUserInfo(token *oauth2.Token) (*UserInfo, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var googleInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&googleInfo); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &UserInfo{
		ID:       googleInfo.ID,
		Email:    googleInfo.Email,
		Name:     googleInfo.Name,
		Verified: googleInfo.VerifiedEmail,
	}, nil
}

// FetchGitHubUserInfo fetches user info from GitHub using the access token
func FetchGitHubUserInfo(token *oauth2.Token) (*UserInfo, error) {
	client := &http.Client{}

	// First fetch user info
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var githubInfo GitHubUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&githubInfo); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// GitHub may not return email in user info if it's private, fetch from emails endpoint
	email := githubInfo.Email
	verified := false

	if email == "" {
		emailsReq, err := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
		if err == nil {
			emailsReq.Header.Set("Authorization", "Bearer "+token.AccessToken)
			emailsReq.Header.Set("Accept", "application/json")

			emailsResp, err := client.Do(emailsReq)
			if err == nil {
				defer emailsResp.Body.Close()

				if emailsResp.StatusCode == http.StatusOK {
					var emails []GitHubUserEmail
					if json.NewDecoder(emailsResp.Body).Decode(&emails) == nil {
						for _, e := range emails {
							if e.Primary {
								email = e.Email
								verified = e.Verified
								break
							}
						}
					}
				}
			}
		}
	}

	return &UserInfo{
		ID:       fmt.Sprintf("%d", githubInfo.ID),
		Email:    email,
		Name:     githubInfo.Name,
		Verified: verified,
	}, nil
}
