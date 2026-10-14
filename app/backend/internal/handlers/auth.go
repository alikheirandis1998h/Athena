package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"base-backend/internal/auth"
	"base-backend/internal/db"
)

type AuthHandler struct {
	queries *db.Queries
	tokens  *auth.TokenManager
}

func NewAuthHandler(queries *db.Queries, tokens *auth.TokenManager) *AuthHandler {
	return &AuthHandler{queries: queries, tokens: tokens}
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type TokenResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	User         UserResponse `json:"user"`
}

// Register godoc
// @Summary      Register a new user
// @Description  Creates a user account and returns an access/refresh token pair
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      RegisterRequest  true  "Registration payload"
// @Success      201      {object}  TokenResponse
// @Failure      400      {object}  map[string]string
// @Failure      409      {object}  map[string]string
// @Router       /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process password"})
		return
	}

	user, err := h.queries.CreateUser(c.Request.Context(), db.CreateUserParams{
		Email:        req.Email,
		PasswordHash: hash,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	tokens, err := h.issueTokenPair(c.Request.Context(), user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue tokens"})
		return
	}

	c.JSON(http.StatusCreated, tokens)
}

// Login godoc
// @Summary      Login
// @Description  Verifies credentials and returns an access/refresh token pair
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      LoginRequest  true  "Login payload"
// @Success      200      {object}  TokenResponse
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Router       /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.queries.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user"})
		return
	}

	if !auth.VerifyPassword(user.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	tokens, err := h.issueTokenPair(c.Request.Context(), user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue tokens"})
		return
	}

	c.JSON(http.StatusOK, tokens)
}

// Refresh godoc
// @Summary      Refresh access token
// @Description  Validates a refresh token and returns a new token pair, rotating the refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      RefreshRequest  true  "Refresh payload"
// @Success      200      {object}  TokenResponse
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Router       /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	claims, err := h.tokens.ParseRefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
		return
	}

	tokenHash := auth.HashToken(req.RefreshToken)
	stored, err := h.queries.GetRefreshTokenByHash(c.Request.Context(), tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token not recognized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate refresh token"})
		return
	}

	if stored.Revoked || stored.ExpiresAt.Time.Before(time.Now()) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token no longer valid"})
		return
	}

	user, err := h.queries.GetUserByID(c.Request.Context(), claims.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	// Rotation-on-use: the presented refresh token is revoked immediately and a brand new
	// one is issued, so a stolen-but-already-used token can never be replayed.
	if err := h.queries.RevokeRefreshToken(c.Request.Context(), stored.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to rotate refresh token"})
		return
	}

	tokens, err := h.issueTokenPair(c.Request.Context(), user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue tokens"})
		return
	}

	c.JSON(http.StatusOK, tokens)
}

// Logout godoc
// @Summary      Logout
// @Description  Revokes the given refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      LogoutRequest  true  "Logout payload"
// @Success      204      "No Content"
// @Failure      400      {object}  map[string]string
// @Router       /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokenHash := auth.HashToken(req.RefreshToken)
	stored, err := h.queries.GetRefreshTokenByHash(c.Request.Context(), tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.Status(http.StatusNoContent)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke refresh token"})
		return
	}

	if err := h.queries.RevokeRefreshToken(c.Request.Context(), stored.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke refresh token"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) issueTokenPair(ctx context.Context, userID uuid.UUID, email string) (TokenResponse, error) {
	accessToken, err := h.tokens.IssueAccessToken(userID, email)
	if err != nil {
		return TokenResponse{}, err
	}

	refreshToken, expiresAt, err := h.tokens.IssueRefreshToken(userID)
	if err != nil {
		return TokenResponse{}, err
	}

	_, err = h.queries.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:    userID,
		TokenHash: auth.HashToken(refreshToken),
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return TokenResponse{}, err
	}

	return TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		User:         UserResponse{ID: userID.String(), Email: email},
	}, nil
}
