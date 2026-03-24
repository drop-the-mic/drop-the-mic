package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// jwtHeader is the fixed HS256 JWT header.
var jwtHeaderEncoded = base64URLEncode([]byte(`{"alg":"HS256","typ":"JWT"}`))

// jwtClaims holds the registered JWT claims used by DTM.
type jwtClaims struct {
	Sub string `json:"sub"`
	Exp int64  `json:"exp"`
	Iat int64  `json:"iat"`
}

// loginRequest is the JSON body expected by the login handler.
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// loginResponse is the JSON body returned on successful login.
type loginResponse struct {
	Token string `json:"token"`
}

// base64URLEncode encodes bytes to unpadded base64url.
func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// base64URLDecode decodes an unpadded base64url string.
func base64URLDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// signJWT creates a minimal HS256 JWT with the given claims and secret.
func signJWT(claims jwtClaims, secret []byte) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}
	payloadEnc := base64URLEncode(payload)
	signingInput := jwtHeaderEncoded + "." + payloadEnc

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signingInput))
	sig := base64URLEncode(mac.Sum(nil))

	return signingInput + "." + sig, nil
}

// verifyJWT parses and validates an HS256 JWT. It returns the claims on success.
func verifyJWT(token string, secret []byte) (jwtClaims, error) {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		return jwtClaims{}, fmt.Errorf("malformed token")
	}

	// Verify signature.
	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signingInput))
	expectedSig := mac.Sum(nil)

	actualSig, err := base64URLDecode(parts[2])
	if err != nil {
		return jwtClaims{}, fmt.Errorf("decode signature: %w", err)
	}

	if !hmac.Equal(expectedSig, actualSig) {
		return jwtClaims{}, fmt.Errorf("invalid signature")
	}

	// Decode payload.
	payload, err := base64URLDecode(parts[1])
	if err != nil {
		return jwtClaims{}, fmt.Errorf("decode payload: %w", err)
	}

	var claims jwtClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return jwtClaims{}, fmt.Errorf("unmarshal claims: %w", err)
	}

	// Check expiry.
	if time.Now().Unix() > claims.Exp {
		return jwtClaims{}, fmt.Errorf("token expired")
	}

	return claims, nil
}

// Login handles POST /api/v1/login. It validates credentials and returns a JWT.
func Login(w http.ResponseWriter, r *http.Request) {
	user := os.Getenv("DTM_AUTH_USER")
	pass := os.Getenv("DTM_AUTH_PASSWORD")

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userMatch := subtle.ConstantTimeCompare([]byte(req.Username), []byte(user)) == 1
	passMatch := subtle.ConstantTimeCompare([]byte(req.Password), []byte(pass)) == 1

	if !userMatch || !passMatch {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	now := time.Now()
	claims := jwtClaims{
		Sub: req.Username,
		Exp: now.Add(24 * time.Hour).Unix(),
		Iat: now.Unix(),
	}

	token, err := signJWT(claims, []byte(pass))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{Token: token})
}

// AuthCheck handles GET /api/v1/auth/check. It validates the JWT and returns 200 if valid.
func AuthCheck(w http.ResponseWriter, r *http.Request) {
	pass := os.Getenv("DTM_AUTH_PASSWORD")

	// If no auth configured, always OK.
	if os.Getenv("DTM_AUTH_USER") == "" && pass == "" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	token := strings.TrimPrefix(header, "Bearer ")
	if _, err := verifyJWT(token, []byte(pass)); err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// JWTAuthMiddleware returns an HTTP middleware that enforces JWT authentication.
// Credentials are read from DTM_AUTH_USER and DTM_AUTH_PASSWORD env vars.
// If both are empty or unset, the middleware is a passthrough.
// Exempt paths: /healthz, /api/v1/login, and all non-/api/ paths (so the UI can load).
func JWTAuthMiddleware(next http.Handler) http.Handler {
	user := os.Getenv("DTM_AUTH_USER")
	pass := os.Getenv("DTM_AUTH_PASSWORD")

	// If credentials are not configured, skip auth entirely.
	if user == "" && pass == "" {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Health check is always public.
		if path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}

		// Login endpoint is always public.
		if path == "/api/v1/login" {
			next.ServeHTTP(w, r)
			return
		}

		// Non-API paths (UI static files) are always public.
		if !strings.HasPrefix(path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		// All other /api/* paths require a valid JWT.
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		token := strings.TrimPrefix(header, "Bearer ")
		if _, err := verifyJWT(token, []byte(pass)); err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		next.ServeHTTP(w, r)
	})
}
