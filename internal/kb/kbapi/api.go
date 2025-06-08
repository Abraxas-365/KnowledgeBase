package kbapi

import (
	"context"
	"strconv"

	kbsrv "github.com/Abraxas-365/opd/internal/kb/kbasesrv"
	"github.com/Abraxas-365/opd/internal/user"
	"github.com/Abraxas-365/opd/pkg/middleware"
	"github.com/Abraxas-365/toolkit/pkg/errors"
	"github.com/gofiber/fiber/v2"
)

// SetupRoutes sets up the API routes for the knowledge base service with paywall integration
func SetupRoutes(app *fiber.App, service *kbsrv.Service, authMiddleware *middleware.JWTAuthMiddleware, paywallLimiter *middleware.PaywallRateLimiter) {
	// Create a group for chat endpoints with paywall rate limiting
	chatGroup := app.Group("/chat")

	// Apply the paywall rate limiting middleware to chat endpoints
	chatGroup.Use(paywallLimiter.RateLimitMiddleware())

	// Define the route for completing answers with metadata (now with paywall protection)
	chatGroup.Post("/complete-answer", func(c *fiber.Ctx) error {
		type Request struct {
			UserMessage string  `json:"userMessage"`
			SessionID   *string `json:"sessionID,omitempty"`
			UserChatID  string  `json:"userChatID,omitempty"`
		}

		var req Request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		// Validate required fields
		if req.UserMessage == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "userMessage is required",
			})
		}

		if req.UserChatID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "userChatID is required",
			})
		}

		output, err := service.CompleteAnswerWithMetadata(context.TODO(), req.UserMessage, req.SessionID, req.UserChatID)
		if err != nil {
			return err
		}

		// Add usage information to response for authenticated users
		authenticated := c.Locals("authenticated")
		if authenticated != nil && authenticated.(bool) {
			// Get rate limit info from headers set by middleware
			limit := c.Get("X-RateLimit-Limit")
			remaining := c.Get("X-RateLimit-Remaining")
			tier := c.Get("X-RateLimit-Tier")

			if limit != "unlimited" {
				// Add usage info to response
				return c.JSON(fiber.Map{
					"answer": output,
					"usage": fiber.Map{
						"requests_limit":     limit,
						"requests_remaining": remaining,
						"tier":               tier,
					},
				})
			}
		}

		return c.JSON(fiber.Map{
			"answer": output,
		})
	})

	// Route to generate a presigned PUT URL (admin/authenticated users only)
	app.Post("/generate-presigned-url", authMiddleware.RequireAuth(), func(c *fiber.Ctx) error {
		type Request struct {
			FileName string `json:"fileName"`
		}

		// Get user ID from context using the JWT middleware
		userID := c.Locals("userID").(string)

		var req Request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		if req.FileName == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "fileName is required",
			})
		}

		url, err := service.GeneratePutURL(userID, req.FileName)
		if err != nil {
			return err
		}

		return c.JSON(fiber.Map{"url": url})
	})

	// Route to list objects with pagination (admin/authenticated users only)
	app.Get("/list-objects", authMiddleware.RequireAuth(), func(c *fiber.Ctx) error {
		type Request struct {
			PageSize          int32  `query:"pageSize"`
			ContinuationToken string `query:"continuationToken"`
		}

		var req Request
		if err := c.QueryParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid query parameters",
			})
		}

		// Set default page size if not provided
		if req.PageSize <= 0 {
			req.PageSize = 10
		}

		var continuationToken *string
		if req.ContinuationToken != "" {
			continuationToken = &req.ContinuationToken
		}

		files, nextToken, err := service.LisObjects(req.PageSize, continuationToken)
		if err != nil {
			return err
		}

		return c.JSON(fiber.Map{
			"files":             files,
			"continuationToken": nextToken,
		})
	})

	// Delete objects (admin only)
	app.Delete("/objects/:id", authMiddleware.RequireAdmin(), func(c *fiber.Ctx) error {
		// Get file id from path parameter
		fileId := c.Params("id")

		// Convert string to int
		id, err := strconv.Atoi(fileId)
		if err != nil {
			return errors.ErrBadRequest("File id must be a number")
		}

		if err := service.DeleteObject(id); err != nil {
			return err
		}

		return c.JSON(fiber.Map{
			"message": "Object deleted successfully",
		})
	})

	// Endpoint to start the ingestion job for syncing knowledge base (admin only)
	app.Post("/sync-knowledge-base", authMiddleware.RequireAdmin(), func(c *fiber.Ctx) error {
		output, err := service.SyncKnowledgeBase(context.TODO())
		if err != nil {
			return err
		}
		return c.JSON(output)
	})

	// Get paginated list of objects (admin/authenticated users only)
	app.Get("/objects", authMiddleware.RequireAuth(), func(c *fiber.Ctx) error {
		// Get page and page size from query parameters
		page, err := strconv.Atoi(c.Query("page", "1"))
		if err != nil || page < 1 {
			return errors.ErrBadRequest("Invalid page number")
		}

		pageSize, err := strconv.Atoi(c.Query("pageSize", "10"))
		if err != nil || pageSize < 1 || pageSize > 100 {
			return errors.ErrBadRequest("Invalid page size (must be between 1 and 100)")
		}

		// Get paginated data from service
		paginatedData, err := service.GetFiles(c.Context(), page, pageSize)
		if err != nil {
			return err
		}

		return c.JSON(paginatedData)
	})

	// Health check endpoint for the chat service
	app.Get("/chat/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "knowledge-base-chat",
		})
	})

	// Rate limit info endpoint for users to check their current usage
	app.Get("/chat/rate-limit-info", authMiddleware.AuthMiddleware(), func(c *fiber.Ctx) error {
		authenticated := c.Locals("authenticated")
		isAuth := authenticated != nil && authenticated.(bool)

		if !isAuth {
			return c.JSON(fiber.Map{
				"user_type":   "anonymous",
				"daily_limit": 5,
				"message":     "Sign up for a free account to get 50 requests per day!",
			})
		}

		userRole := c.Locals("role")
		if userRole != nil && userRole.(user.Role) == user.RoleAdmin {
			return c.JSON(fiber.Map{
				"user_type":   "admin",
				"daily_limit": "unlimited",
				"message":     "You have unlimited access as an admin user.",
			})
		}

		subscription := c.Locals("subscriptionType")
		if subscription == nil {
			return c.JSON(fiber.Map{
				"user_type":   "unknown",
				"daily_limit": 0,
				"message":     "Unable to determine subscription type.",
			})
		}

		userSub := subscription.(user.SubscriptionType)
		var limit int
		var tierName string
		var upgradeMessage string

		switch userSub {
		case user.FreeTier:
			limit = 50
			tierName = "Free Tier"
			upgradeMessage = "Upgrade to Linko Plus for 200 requests per day!"
		case user.LinkoPlus:
			limit = 200
			tierName = "Linko Plus"
			upgradeMessage = "Upgrade to Linko VIP for 1000 requests per day!"
		case user.LinkoVIP:
			limit = 1000
			tierName = "Linko VIP"
			upgradeMessage = "You have our highest tier with maximum daily requests!"
		default:
			limit = 0
			tierName = "Unknown"
			upgradeMessage = "Please contact support for subscription assistance."
		}

		return c.JSON(fiber.Map{
			"user_type":         "authenticated",
			"subscription_tier": tierName,
			"daily_limit":       limit,
			"upgrade_message":   upgradeMessage,
		})
	})
}
