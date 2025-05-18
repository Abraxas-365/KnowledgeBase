package analiticsapi

import (
	"time"

	"github.com/Abraxas-365/opd/internal/analitics/analiticssrv"
	"github.com/Abraxas-365/opd/pkg/middleware"
	"github.com/Abraxas-365/toolkit/pkg/errors"
	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(
	app *fiber.App,
	service *analiticssrv.Service,
	authMiddleware *middleware.JWTAuthMiddleware,
) {
	app.Get("/analytics", authMiddleware.RequireAuth(), getAnalytics(service))
	app.Get("/analytics/daily/users", authMiddleware.RequireAuth(), getDailyUsers(service))
	app.Get("/analytics/daily/interactions", authMiddleware.RequireAuth(), getDailyInteractions(service))
	app.Get("/analytics/export", authMiddleware.RequireAuth(), exportDatabase(service))
}

func getAnalytics(service *analiticssrv.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Parse query parameters for date range
		startDateStr := c.Query("start_date")
		endDateStr := c.Query("end_date")

		var startDate, endDate *time.Time

		// Parse start date if provided
		if startDateStr != "" {
			parsedDate, err := time.Parse("2006-01-02", startDateStr)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Invalid start_date format. Use YYYY-MM-DD",
				})
			}
			startDate = &parsedDate
		}

		// Parse end date if provided
		if endDateStr != "" {
			parsedDate, err := time.Parse("2006-01-02", endDateStr)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Invalid end_date format. Use YYYY-MM-DD",
				})
			}
			endDate = &parsedDate
		}

		// Validate date range if both dates are provided
		if startDate != nil && endDate != nil && endDate.Before(*startDate) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "end_date cannot be before start_date",
			})
		}

		// Get analytics
		analytics, err := service.GetAllAnalitics(c.Context(), startDate, endDate)
		if err != nil {
			switch {
			case errors.IsNotFound(err):
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": err.Error(),
				})
			case errors.IsDatabaseError(err):
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Database error occurred",
				})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to fetch analytics",
				})
			}
		}

		return c.JSON(fiber.Map{
			"data": analytics,
		})
	}
}

func getDailyUsers(service *analiticssrv.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		startDateStr := c.Query("start_date")
		endDateStr := c.Query("end_date")

		if startDateStr == "" || endDateStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Both start_date and end_date are required",
			})
		}

		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid start_date format. Use YYYY-MM-DD",
			})
		}

		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid end_date format. Use YYYY-MM-DD",
			})
		}

		if endDate.Before(startDate) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "end_date cannot be before start_date",
			})
		}

		dailyStats, err := service.GetDailyUsersInRange(c.Context(), startDate, endDate)
		if err != nil {
			switch {
			case errors.IsNotFound(err):
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": err.Error(),
				})
			case errors.IsDatabaseError(err):
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Database error occurred",
				})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to fetch daily user statistics",
				})
			}
		}

		return c.JSON(fiber.Map{
			"data": dailyStats,
		})
	}
}

func getDailyInteractions(service *analiticssrv.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		startDateStr := c.Query("start_date")
		endDateStr := c.Query("end_date")

		if startDateStr == "" || endDateStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Both start_date and end_date are required",
			})
		}

		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid start_date format. Use YYYY-MM-DD",
			})
		}

		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid end_date format. Use YYYY-MM-DD",
			})
		}

		if endDate.Before(startDate) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "end_date cannot be before start_date",
			})
		}

		dailyStats, err := service.GetDailyInteractionsInRange(c.Context(), startDate, endDate)
		if err != nil {
			switch {
			case errors.IsNotFound(err):
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": err.Error(),
				})
			case errors.IsDatabaseError(err):
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Database error occurred",
				})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to fetch daily interaction statistics",
				})
			}
		}

		return c.JSON(fiber.Map{
			"data": dailyStats,
		})
	}
}

func exportDatabase(service *analiticssrv.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Parse query parameters for date range
		startDateStr := c.Query("start_date")
		endDateStr := c.Query("end_date")

		var startDate, endDate *time.Time

		// Parse start date if provided
		if startDateStr != "" {
			parsedDate, err := time.Parse("2006-01-02", startDateStr)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Invalid start_date format. Use YYYY-MM-DD",
				})
			}
			startDate = &parsedDate
		}

		// Parse end date if provided
		if endDateStr != "" {
			parsedDate, err := time.Parse("2006-01-02", endDateStr)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Invalid end_date format. Use YYYY-MM-DD",
				})
			}
			endDate = &parsedDate
		}

		// Validate date range if both dates are provided
		if startDate != nil && endDate != nil && endDate.Before(*startDate) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "end_date cannot be before start_date",
			})
		}

		// Export database to CSV
		presignedURL, err := service.ExportDatabaseToCSV(c.Context(), startDate, endDate)
		if err != nil {
			switch {
			case errors.IsNotFound(err):
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": err.Error(),
				})
			case errors.IsServiceUnavailable(err):
				return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
					"error": err.Error(),
				})
			case errors.IsDatabaseError(err):
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Database error occurred",
				})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to export database",
				})
			}
		}

		return c.JSON(fiber.Map{
			"data": fiber.Map{
				"download_url": presignedURL,
			},
		})
	}
}
