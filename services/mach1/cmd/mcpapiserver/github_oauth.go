package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/clouddb"
)

type githubOAuthState struct {
	mu     sync.Mutex
	states map[string]time.Time
}

func newGitHubOAuthState() *githubOAuthState {
	return &githubOAuthState{
		states: make(map[string]time.Time),
	}
}

func (s *githubOAuthState) generateState() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	state := fmt.Sprintf("%x", b)
	s.mu.Lock()
	s.states[state] = time.Now().Add(10 * time.Minute)
	s.mu.Unlock()
	return state
}

func (s *githubOAuthState) validateState(state string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.states[state]
	if !ok {
		return false
	}
	delete(s.states, state)
	return time.Now().Before(exp)
}

func registerGitHubOAuthHandlers(mux *http.ServeMux, db *clouddb.DB) {
	stateStore := newGitHubOAuthState()
	mux.HandleFunc("GET /api/auth/github/url", handleGitHubAuthURL(stateStore))
	mux.HandleFunc("POST /api/auth/github/exchange", handleGitHubAuthExchange(db, stateStore))
}

func handleGitHubAuthURL(store *githubOAuthState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientID := os.Getenv("GITHUB_CLIENT_ID")
		if clientID == "" {
			writeJSON(w, http.StatusInternalServerError, errBody("GitHub OAuth not configured"))
			return
		}

		redirectURI := r.URL.Query().Get("redirect_uri")
		if redirectURI == "" {
			writeJSON(w, http.StatusBadRequest, errBody("redirect_uri is required"))
			return
		}

		state := store.generateState()

		u := url.URL{
			Scheme: "https",
			Host:   "github.com",
			Path:   "/login/oauth/authorize",
		}
		q := u.Query()
		q.Set("client_id", clientID)
		q.Set("redirect_uri", redirectURI)
		q.Set("state", state)
		q.Set("scope", "user:email")
		u.RawQuery = q.Encode()

		writeJSON(w, http.StatusOK, map[string]string{"url": u.String()})
	}
}

func handleGitHubAuthExchange(db *clouddb.DB, stateStore *githubOAuthState) http.HandlerFunc {
	type request struct {
		Code        string `json:"code"`
		RedirectURI string `json:"redirect_uri"`
		State       string `json:"state,omitempty"`
	}
	type githubTokenResponse struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}
	type githubUser struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	type githubEmail struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var body request
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("invalid request body"))
			return
		}
		if body.Code == "" || body.RedirectURI == "" {
			writeJSON(w, http.StatusBadRequest, errBody("code and redirect_uri are required"))
			return
		}

		if body.State != "" && !stateStore.validateState(body.State) {
			writeJSON(w, http.StatusBadRequest, errBody("invalid or expired state"))
			return
		}

		clientID := os.Getenv("GITHUB_CLIENT_ID")
		clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
		if clientID == "" || clientSecret == "" {
			writeJSON(w, http.StatusInternalServerError, errBody("GitHub OAuth not configured"))
			return
		}

		// Exchange code for access token
		tokenURL := "https://github.com/login/oauth/access_token"
		req, err := http.NewRequest("POST", tokenURL, strings.NewReader(url.Values{
			"client_id":     {clientID},
			"client_secret": {clientSecret},
			"code":          {body.Code},
			"redirect_uri":  {body.RedirectURI},
		}.Encode()))
		if err != nil {
			slog.Error("create github token request", "err", err)
			writeJSON(w, http.StatusInternalServerError, errBody("failed to exchange code"))
			return
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			slog.Error("github token exchange", "err", err)
			writeJSON(w, http.StatusInternalServerError, errBody("failed to exchange code"))
			return
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			slog.Error("read github token response", "err", err)
			writeJSON(w, http.StatusInternalServerError, errBody("failed to read token response"))
			return
		}

		var tokenResp githubTokenResponse
		if err := json.Unmarshal(respBody, &tokenResp); err != nil {
			// Fallback to form-encoded parsing
			values, err := url.ParseQuery(string(respBody))
			if err != nil {
				slog.Error("parse github token response", "err", err)
				writeJSON(w, http.StatusInternalServerError, errBody("failed to parse token response"))
				return
			}
			tokenResp.AccessToken = values.Get("access_token")
			tokenResp.TokenType = values.Get("token_type")
			tokenResp.Scope = values.Get("scope")
		}

		if tokenResp.AccessToken == "" {
			slog.Error("github token empty", "response", string(respBody))
			writeJSON(w, http.StatusUnauthorized, errBody("invalid code or credentials"))
			return
		}

		// Fetch user info
		userReq, err := http.NewRequest("GET", "https://api.github.com/user", nil)
		if err != nil {
			slog.Error("create github user request", "err", err)
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}
		userReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
		userReq.Header.Set("Accept", "application/vnd.github+json")

		userResp, err := http.DefaultClient.Do(userReq)
		if err != nil {
			slog.Error("github user request", "err", err)
			writeJSON(w, http.StatusInternalServerError, errBody("failed to fetch user info"))
			return
		}
		defer userResp.Body.Close()

		var ghUser githubUser
		if err := json.NewDecoder(userResp.Body).Decode(&ghUser); err != nil {
			slog.Error("decode github user", "err", err)
			writeJSON(w, http.StatusInternalServerError, errBody("failed to parse user info"))
			return
		}

		email := ghUser.Email
		name := ghUser.Name
		if name == "" {
			name = ghUser.Login
		}

		// If no public email, fetch emails endpoint
		if email == "" {
			emailReq, err := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
			if err != nil {
				slog.Error("create github emails request", "err", err)
				writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
				return
			}
			emailReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
			emailReq.Header.Set("Accept", "application/vnd.github+json")

			emailResp, err := http.DefaultClient.Do(emailReq)
			if err != nil {
				slog.Error("github emails request", "err", err)
				writeJSON(w, http.StatusInternalServerError, errBody("failed to fetch emails"))
				return
			}
			defer emailResp.Body.Close()

			var emails []githubEmail
			if err := json.NewDecoder(emailResp.Body).Decode(&emails); err != nil {
				slog.Error("decode github emails", "err", err)
				writeJSON(w, http.StatusInternalServerError, errBody("failed to parse emails"))
				return
			}

			for _, e := range emails {
				if e.Primary && e.Verified {
					email = e.Email
					break
				}
			}
			if email == "" && len(emails) > 0 {
				email = emails[0].Email
			}
		}

		if email == "" {
			writeJSON(w, http.StatusBadRequest, errBody("GitHub account has no email"))
			return
		}

		// Find or create user
		ctx := r.Context()
		u, err := db.FindUserByEmail(ctx, email)
		if err != nil {
			// Create new user with random password
			randomPass := make([]byte, 32)
			_, _ = rand.Read(randomPass)
			hash, err := bcrypt.GenerateFromPassword(randomPass, bcrypt.DefaultCost)
			if err != nil {
				slog.Error("generate password hash", "err", err)
				writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
				return
			}

			_, err = db.RegisterUser(ctx, email, name, string(hash))
			if err != nil {
				if strings.Contains(err.Error(), "already registered") {
					// Race condition — fetch the existing user
					u, err = db.FindUserByEmail(ctx, email)
					if err != nil {
						slog.Error("find user after register conflict", "err", err)
						writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
						return
					}
				} else {
					slog.Error("register github user", "err", err)
					writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
					return
				}
			} else {
				u, err = db.FindUserByEmail(ctx, email)
				if err != nil {
					slog.Error("find newly registered user", "err", err)
					writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
					return
				}
			}
		}

		token, err := db.CreateSession(ctx, u.ID)
		if err != nil {
			slog.Error("create session", "err", err)
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"token": token,
			"user": map[string]string{
				"id":       u.ID,
				"name":     u.Name,
				"email":    u.Email,
				"username": u.Username,
			},
		})
	}
}
