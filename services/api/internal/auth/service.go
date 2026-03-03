package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/wick/github-star-manager/services/api/internal/db"
	"github.com/wick/github-star-manager/services/api/internal/github"
	"github.com/wick/github-star-manager/services/api/internal/security"
)

type Service struct {
	repo         *db.Repository
	ghClient     *github.Client
	tokenCipher  *security.TokenCipher
	sessionCache SessionCache
}

type LoginResult struct {
	Token   string         `json:"token"`
	Profile db.UserProfile `json:"profile"`
}

func NewService(repo *db.Repository, ghClient *github.Client, tokenCipher *security.TokenCipher, sessionCache SessionCache) *Service {
	return &Service{repo: repo, ghClient: ghClient, tokenCipher: tokenCipher, sessionCache: sessionCache}
}

func (s *Service) LoginWithGitHubToken(ctx context.Context, githubToken string) (LoginResult, error) {
	if githubToken == "" {
		return LoginResult{}, fmt.Errorf("github token is required")
	}

	user, err := s.ghClient.GetAuthenticatedUser(ctx, githubToken)
	if err != nil {
		return LoginResult{}, fmt.Errorf("get github user: %w", err)
	}

	displayName := user.Name
	if displayName == "" {
		displayName = user.Login
	}

	userID, err := s.repo.EnsureUserByGitHubID(ctx, user.ID, displayName)
	if err != nil {
		return LoginResult{}, fmt.Errorf("ensure user: %w", err)
	}

	encryptedToken, err := s.tokenCipher.Encrypt(githubToken)
	if err != nil {
		return LoginResult{}, fmt.Errorf("encrypt token: %w", err)
	}

	if err := s.repo.SaveGitHubAccount(ctx, userID, user.ID, user.Login, user.AvatarURL, encryptedToken, "read:user,repo"); err != nil {
		return LoginResult{}, fmt.Errorf("save github account: %w", err)
	}

	sessionToken := uuid.NewString() + uuid.NewString()
	hash := sha256.Sum256([]byte(sessionToken))
	hashedToken := hex.EncodeToString(hash[:])

	sessionTTL := 24 * time.Hour
	if err := s.repo.CreateSession(ctx, uuid.NewString(), userID, hashedToken, time.Now().Add(sessionTTL)); err != nil {
		return LoginResult{}, fmt.Errorf("create session: %w", err)
	}
	if s.sessionCache != nil {
		_ = s.sessionCache.SetUserIDByTokenHash(ctx, hashedToken, userID, sessionTTL)
	}

	return LoginResult{
		Token: sessionToken,
		Profile: db.UserProfile{
			ID:          userID,
			DisplayName: displayName,
			GitHubLogin: user.Login,
			AvatarURL:   user.AvatarURL,
		},
	}, nil
}

func (s *Service) ValidateSessionToken(ctx context.Context, token string) (int64, error) {
	if token == "" {
		return 0, fmt.Errorf("token is required")
	}
	hash := sha256.Sum256([]byte(token))
	hashedToken := hex.EncodeToString(hash[:])

	if s.sessionCache != nil {
		cachedUserID, found, err := s.sessionCache.GetUserIDByTokenHash(ctx, hashedToken)
		if err != nil {
			return 0, fmt.Errorf("get cached session: %w", err)
		}
		if found {
			return cachedUserID, nil
		}
	}

	userID, err := s.repo.ValidateSession(ctx, hashedToken)
	if err != nil {
		return 0, fmt.Errorf("validate session: %w", err)
	}
	if s.sessionCache != nil {
		_ = s.sessionCache.SetUserIDByTokenHash(ctx, hashedToken, userID, 5*time.Minute)
	}
	return userID, nil
}

func (s *Service) GetGitHubAccessToken(ctx context.Context, userID int64) (string, error) {
	_, _, _, tokenEncrypted, err := s.repo.GetGitHubAccountByUserID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("get github account: %w", err)
	}
	plain, err := s.tokenCipher.Decrypt(tokenEncrypted)
	if err != nil {
		return "", fmt.Errorf("decrypt token: %w", err)
	}
	return plain, nil
}
