package kbapi

import (
	"context"
	"strconv"
	"time"

	kbsrv "github.com/Abraxas-365/opd/internal/kb/kbasesrv"
	"github.com/Abraxas-365/opd/pkg/middleware" // Updated import
	"github.com/Abraxas-365/toolkit/pkg/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// SetupRoutes sets up the API routes for the knowledge base service
func SetupRoutes(app *fiber.App, service *kbsrv.Service, authMiddleware *middleware.JWTAuthMiddleware) { // Changed type here
	// Define the route for completing answers with metadata
	limiterGroup := app.Group("/chat", limiter.New(limiter.Config{
		Max:        100,           // Maximum number of requests per IP
		Expiration: 8 * time.Hour, // Time window for rate limiting
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP() // Limit by IP
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).SendString("Rate limit exceeded")
		},
	}))
	limiterGroup.Post("/complete-answer", func(c *fiber.Ctx) error {
		type Request struct {
			UserMessage string  `json:"userMessage"`
			SessionID   *string `json:"sessionID,omitempty"`
			UserChatID  string  `json:"userChatID,omitempty"`
		}

		var req Request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		output, err := service.CompleteAnswerWithMetadata(context.TODO(), req.UserMessage, req.SessionID, req.UserChatID)
		if err != nil {
			return err
		}

		return c.JSON(output)
	})

	// Route to generate a presigned PUT URL
	app.Post("/generate-presigned-url", authMiddleware.RequireAuth(), func(c *fiber.Ctx) error {
		type Request struct {
			FileName string `json:"fileName"`
		}

		// Get user ID from context using the JWT middleware
		userID := c.Locals("userID").(string)

		var req Request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		url, err := service.GeneratePutURL(userID, req.FileName)
		if err != nil {
			return err
		}

		return c.JSON(fiber.Map{"url": url})
	})

	// Route to list objects with pagination
	app.Get("/list-objects", authMiddleware.RequireAuth(), func(c *fiber.Ctx) error {
		type Request struct {
			PageSize          int32  `query:"pageSize"`
			ContinuationToken string `query:"continuationToken"`
		}

		var req Request
		if err := c.QueryParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid query parameters"})
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

	app.Delete("/objects/:id", authMiddleware.RequireAuth(), func(c *fiber.Ctx) error {
		// Check if user is admin
		isAdmin := c.Locals("isAdmin")
		if isAdmin == nil || !isAdmin.(bool) {
			return errors.ErrForbidden("Admin access required")
		}

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

		return c.JSON(fiber.Map{"message": "Object deleted successfully"})
	})

	// Endpoint to start the ingestion job for syncing knowledge base
	app.Post("/sync-knowledge-base", authMiddleware.RequireAuth(), func(c *fiber.Ctx) error {
		// Check if user is admin
		isAdmin := c.Locals("isAdmin")
		if isAdmin == nil || !isAdmin.(bool) {
			return errors.ErrForbidden("Admin access required")
		}

		output, err := service.SyncKnowledgeBase(context.TODO())
		if err != nil {
			return err
		}
		return c.JSON(output)
	})

	app.Get("/objects", authMiddleware.RequireAuth(), func(c *fiber.Ctx) error {
		// Get page and page size from query parameters
		page, err := strconv.Atoi(c.Query("page", "1"))
		if err != nil || page < 1 {
			return errors.ErrBadRequest("Invalid page number")
		}

		pageSize, err := strconv.Atoi(c.Query("pageSize", "10"))
		if err != nil || pageSize < 1 {
			return errors.ErrBadRequest("Invalid page size")
		}

		// Get paginated data from service
		paginatedData, err := service.GetFiles(c.Context(), page, pageSize)
		if err != nil {
			return err
		}

		return c.JSON(paginatedData)
	})

	// TODO: Implement the logic to get the status of the ingestion job
	// Endpoint to get the status of the ingestion job
	//
	//	app.Get("/ingestion-job-status", authMiddleware.RequireAuth(), func(c *fiber.Ctx) error {
	//		type Request struct {
	//			IngestionJobId string `query:"ingestionJobId"`
	//		}
	//
	//		var req Request
	//		if err := c.QueryParser(&req); err != nil {
	//			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid query parameters"})
	//		}
	//
	//		// Implement call to check the ingestion job status using service.GetIngestionJobStatus (to be created in Service)
	//		status, err := service.GetIngestionJobStatus(req.IngestionJobId)
	//		if err != nil {
	//			return err
	//		}
	//
	//		return c.JSON(fiber.Map{
	//			"status": status,
	//		})
	//	})
}
