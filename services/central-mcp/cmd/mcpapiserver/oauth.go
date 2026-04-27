package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type oauthStore struct {
	mu      sync.Mutex
	clients map[string]oauthClient
	codes   map[string]oauthCode
	tokens  map[string]oauthToken
}

type oauthClient struct {
	ID           string   `json:"client_id"`
	RedirectURIs []string `json:"redirect_uris"`
	Scope        string   `json:"scope,omitempty"`
	CreatedAt    int64    `json:"created_at"`
}

type oauthCode struct {
	ClientID      string
	RedirectURI   string
	Scope         string
	Resource      string
	CodeChallenge string
	ExpiresAt     time.Time
}

type oauthToken struct {
	AccessToken string
	ClientID    string
	Scope       string
	Resource    string
	ExpiresAt   time.Time
}

func newOAuthStore() *oauthStore {
	return &oauthStore{clients: map[string]oauthClient{}, codes: map[string]oauthCode{}, tokens: map[string]oauthToken{}}
}

func registerOAuthHandlers(mux *http.ServeMux, store *oauthStore, issuer string) {
	mux.HandleFunc("GET /.well-known/oauth-authorization-server", store.handleAuthorizationServerMetadata(issuer))
	mux.HandleFunc("GET /.well-known/oauth-protected-resource", store.handleProtectedResourceMetadata(issuer))
	mux.HandleFunc("POST /oauth/register", store.handleDynamicClientRegistration)
	mux.HandleFunc("GET /oauth/authorize", store.handleAuthorize)
	mux.HandleFunc("POST /oauth/token", store.handleToken)
	mux.HandleFunc("POST /oauth/introspect", store.handleIntrospect)
}

func (s *oauthStore) handleAuthorizationServerMetadata(issuer string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"issuer":                                issuer,
			"authorization_endpoint":                issuer + "/oauth/authorize",
			"token_endpoint":                        issuer + "/oauth/token",
			"registration_endpoint":                 issuer + "/oauth/register",
			"introspection_endpoint":                issuer + "/oauth/introspect",
			"code_challenge_methods_supported":      []string{"S256"},
			"grant_types_supported":                 []string{"authorization_code"},
			"response_types_supported":              []string{"code"},
			"scopes_supported":                      []string{"mcp:read", "mcp:write"},
			"resource_indicators_supported":         true,
			"token_endpoint_auth_methods_supported": []string{"none"},
		})
	}
}

func (s *oauthStore) handleProtectedResourceMetadata(issuer string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resource := r.URL.Query().Get("resource")
		if resource == "" {
			resource = issuer + "/mcp"
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"resource":                     resource,
			"authorization_servers":        []string{issuer},
			"scopes_supported":             []string{"mcp:read", "mcp:write"},
			"bearer_methods_supported":     []string{"header"},
			"resource_documentation":       "https://1mcp.in/docs/oauth",
			"resource_indicators_required": true,
		})
	}
}

func (s *oauthStore) handleDynamicClientRegistration(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RedirectURIs []string `json:"redirect_uris"`
		Scope        string   `json:"scope"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errBody("invalid request body"))
		return
	}
	if len(req.RedirectURIs) == 0 {
		writeJSON(w, http.StatusBadRequest, errBody("redirect_uris required"))
		return
	}
	if !validScopes(req.Scope) {
		writeJSON(w, http.StatusBadRequest, errBody("scope must be mcp:read, mcp:write, or both"))
		return
	}
	client := oauthClient{ID: randomToken(24), RedirectURIs: req.RedirectURIs, Scope: req.Scope, CreatedAt: time.Now().Unix()}
	s.mu.Lock()
	s.clients[client.ID] = client
	s.mu.Unlock()
	writeJSON(w, http.StatusCreated, client)
}

func (s *oauthStore) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("response_type") != "code" {
		writeJSON(w, http.StatusBadRequest, errBody("response_type must be code"))
		return
	}
	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")
	scope := q.Get("scope")
	resource := q.Get("resource")
	challenge := q.Get("code_challenge")
	if q.Get("code_challenge_method") != "S256" || challenge == "" {
		writeJSON(w, http.StatusBadRequest, errBody("PKCE S256 code_challenge required"))
		return
	}
	if !validScopes(scope) || !validResource(resource) {
		writeJSON(w, http.StatusBadRequest, errBody("valid scope and resource indicator are required"))
		return
	}
	s.mu.Lock()
	client, ok := s.clients[clientID]
	s.mu.Unlock()
	if !ok || !contains(client.RedirectURIs, redirectURI) {
		writeJSON(w, http.StatusBadRequest, errBody("unknown client or redirect_uri"))
		return
	}
	code := randomToken(32)
	s.mu.Lock()
	s.codes[code] = oauthCode{ClientID: clientID, RedirectURI: redirectURI, Scope: scope, Resource: resource, CodeChallenge: challenge, ExpiresAt: time.Now().Add(10 * time.Minute)}
	s.mu.Unlock()
	state := q.Get("state")
	redirect, _ := url.Parse(redirectURI)
	values := redirect.Query()
	values.Set("code", code)
	if state != "" {
		values.Set("state", state)
	}
	redirect.RawQuery = values.Encode()
	http.Redirect(w, r, redirect.String(), http.StatusFound)
}

func (s *oauthStore) handleToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, errBody("invalid form body"))
		return
	}
	if r.Form.Get("grant_type") != "authorization_code" {
		writeJSON(w, http.StatusBadRequest, errBody("grant_type must be authorization_code"))
		return
	}
	code := r.Form.Get("code")
	verifier := r.Form.Get("code_verifier")
	resource := r.Form.Get("resource")
	s.mu.Lock()
	oc, ok := s.codes[code]
	if ok {
		delete(s.codes, code)
	}
	s.mu.Unlock()
	if !ok || time.Now().After(oc.ExpiresAt) || resource != oc.Resource || !pkceMatches(verifier, oc.CodeChallenge) {
		writeJSON(w, http.StatusUnauthorized, errBody("invalid authorization code, verifier, or resource"))
		return
	}
	token := randomToken(32)
	ot := oauthToken{AccessToken: token, ClientID: oc.ClientID, Scope: oc.Scope, Resource: oc.Resource, ExpiresAt: time.Now().Add(time.Hour)}
	s.mu.Lock()
	s.tokens[token] = ot
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   3600,
		"scope":        oc.Scope,
		"resource":     oc.Resource,
	})
}

func (s *oauthStore) handleIntrospect(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, errBody("invalid form body"))
		return
	}
	token := r.Form.Get("token")
	s.mu.Lock()
	ot, ok := s.tokens[token]
	s.mu.Unlock()
	active := ok && time.Now().Before(ot.ExpiresAt)
	writeJSON(w, http.StatusOK, map[string]any{
		"active":    active,
		"client_id": ot.ClientID,
		"scope":     ot.Scope,
		"resource":  ot.Resource,
		"exp":       ot.ExpiresAt.Unix(),
	})
}

func validScopes(scope string) bool {
	if scope == "" {
		return false
	}
	for _, part := range strings.Fields(scope) {
		if part != "mcp:read" && part != "mcp:write" {
			return false
		}
	}
	return true
}

func validResource(resource string) bool {
	if resource == "" {
		return false
	}
	u, err := url.Parse(resource)
	if err != nil || u.Host == "" {
		return false
	}
	return u.Scheme == "https" || strings.HasPrefix(u.Host, "localhost") || strings.HasPrefix(u.Host, "127.0.0.1")
}

func pkceMatches(verifier, challenge string) bool {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:]) == challenge
}

func randomToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
