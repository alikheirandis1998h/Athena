package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"base-backend/internal/auth"
	"base-backend/internal/db"
)

type UserHandler struct {
	queries *db.Queries
}

func NewUserHandler(queries *db.Queries) *UserHandler {
	return &UserHandler{queries: queries}
}

type UserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// Me godoc
// @Summary      Get current user
// @Description  Returns the authenticated user's profile
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  UserResponse
// @Failure      401  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /api/v1/users/me [get]
func (h *UserHandler) Me(c *gin.Context) {
	claims, ok := auth.ClaimsFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, err := h.queries.GetUserByID(c.Request.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user"})
		return
	}

	c.JSON(http.StatusOK, UserResponse{
		ID:    user.ID.String(),
		Email: user.Email,
	})
}
