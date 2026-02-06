package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ResourceOwnerChecker is an interface for checking resource ownership
type ResourceOwnerChecker interface {
	IsResourceOwner(ctx context.Context, userID, resourceID string) (bool, error)
}

// AuthorizeResource creates a middleware that checks if the authenticated user owns a resource
func AuthorizeResource(checker ResourceOwnerChecker, resourceIDParam string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get authenticated user ID
		userID, exists := GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authentication required",
			})
			c.Abort()
			return
		}

		// Get resource ID from URL parameter
		resourceID := c.Param(resourceIDParam)
		if resourceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "resource ID required",
			})
			c.Abort()
			return
		}

		// Check ownership
		isOwner, err := checker.IsResourceOwner(c.Request.Context(), userID, resourceID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to verify resource ownership",
			})
			c.Abort()
			return
		}

		if !isOwner {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "access denied: you do not own this resource",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ChatSessionOwnerChecker checks if a user owns a chat session
type ChatSessionOwnerChecker struct {
	getSessionOwner func(ctx context.Context, sessionID string) (string, error)
}

// NewChatSessionOwnerChecker creates a new chat session owner checker
func NewChatSessionOwnerChecker(getSessionOwner func(ctx context.Context, sessionID string) (string, error)) *ChatSessionOwnerChecker {
	return &ChatSessionOwnerChecker{
		getSessionOwner: getSessionOwner,
	}
}

// IsResourceOwner checks if the user owns the chat session
func (c *ChatSessionOwnerChecker) IsResourceOwner(ctx context.Context, userID, resourceID string) (bool, error) {
	ownerID, err := c.getSessionOwner(ctx, resourceID)
	if err != nil {
		return false, err
	}
	return ownerID == userID, nil
}

// ResearchSessionOwnerChecker checks if a user owns a research session
type ResearchSessionOwnerChecker struct {
	getSessionOwner func(ctx context.Context, sessionID string) (string, error)
}

// NewResearchSessionOwnerChecker creates a new research session owner checker
func NewResearchSessionOwnerChecker(getSessionOwner func(ctx context.Context, sessionID string) (string, error)) *ResearchSessionOwnerChecker {
	return &ResearchSessionOwnerChecker{
		getSessionOwner: getSessionOwner,
	}
}

// IsResourceOwner checks if the user owns the research session
func (r *ResearchSessionOwnerChecker) IsResourceOwner(ctx context.Context, userID, resourceID string) (bool, error) {
	ownerID, err := r.getSessionOwner(ctx, resourceID)
	if err != nil {
		return false, err
	}
	return ownerID == userID, nil
}
