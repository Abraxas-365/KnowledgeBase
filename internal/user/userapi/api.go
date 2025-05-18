package userapi

import (
	"context"
	"fmt"
	"log"

	"github.com/Abraxas-365/opd/internal/user"
	"github.com/Abraxas-365/opd/internal/user/usersrv"
	"github.com/Abraxas-365/opd/pkg/middleware" // Updated import
	"github.com/gofiber/fiber/v2"
)

// SetupRoutes sets up the API routes for the user service
func SetupRoutes(app *fiber.App, service *usersrv.Service, authMiddleware *middleware.JWTAuthMiddleware) { // Changed type here
	app.Get("/users/me", authMiddleware.RequireAuth(), func(c *fiber.Ctx) error {
		log.Println("Accessing /users/me endpoint")

		// Use userID from JWT token stored in context
		userID := c.Locals("userID").(string)

		user, err := service.GetUser(c.Context(), userID)
		if err != nil {
			log.Printf("Error fetching user details: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Failed to fetch user details: %v", err)})
		}

		log.Println("Successfully fetched user details")
		return c.JSON(user)
	})

	// Require admin role for these endpoints
	app.Get("/users", authMiddleware.RequireAdmin(), func(c *fiber.Ctx) error {
		page, pageSize := 1, 10 // Default values
		pageParam := c.Query("page")
		pageSizeParam := c.Query("pageSize")

		// Parse pagination params if provided
		if pageParam != "" {
			page = c.QueryInt("page")
		}
		if pageSizeParam != "" {
			pageSize = c.QueryInt("pageSize")
		}

		users, err := service.GetUsers(context.TODO(), page, pageSize)
		if err != nil {
			return err
		}

		return c.JSON(users)
	})

	app.Get("/users/not-admin", authMiddleware.RequireAdmin(), func(c *fiber.Ctx) error {
		page, pageSize := 1, 10
		if p := c.Query("page"); p != "" {
			page = c.QueryInt("page")
		}
		if ps := c.Query("pageSize"); ps != "" {
			pageSize = c.QueryInt("pageSize")
		}

		users, err := service.GetNotAdminUsers(context.TODO(), page, pageSize)
		if err != nil {
			return err
		}

		return c.JSON(users)
	})

	app.Get("/users/admin", authMiddleware.RequireAdmin(), func(c *fiber.Ctx) error {
		page, pageSize := 1, 10
		if p := c.Query("page"); p != "" {
			page = c.QueryInt("page")
		}
		if ps := c.Query("pageSize"); ps != "" {
			pageSize = c.QueryInt("pageSize")
		}

		users, err := service.GetUsersAdminRole(context.TODO(), page, pageSize)
		if err != nil {
			return err
		}

		return c.JSON(users)
	})

	app.Get("/users/role/:role", authMiddleware.RequireAdmin(), func(c *fiber.Ctx) error {
		roleParam := c.Params("role")
		var role user.Role

		switch roleParam {
		case "admin":
			role = user.RoleAdmin
		case "linko-user":
			role = user.RoleLinkoUser
		default:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid role"})
		}

		page, pageSize := 1, 10
		if p := c.Query("page"); p != "" {
			page = c.QueryInt("page")
		}
		if ps := c.Query("pageSize"); ps != "" {
			pageSize = c.QueryInt("pageSize")
		}

		users, err := service.GetUsersByRole(context.TODO(), role, page, pageSize)
		if err != nil {
			return err
		}

		return c.JSON(users)
	})

	app.Get("/users/subscription/:type", authMiddleware.RequireAdmin(), func(c *fiber.Ctx) error {
		typeParam := c.Params("type")
		var subType user.SubscriptionType

		switch typeParam {
		case "free-tier":
			subType = user.FreeTier
		case "linko-plus":
			subType = user.LinkoPlus
		case "linko-vip":
			subType = user.LinkoVIP
		default:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid subscription type"})
		}

		page, pageSize := 1, 10
		if p := c.Query("page"); p != "" {
			page = c.QueryInt("page")
		}
		if ps := c.Query("pageSize"); ps != "" {
			pageSize = c.QueryInt("pageSize")
		}

		users, err := service.GetUsersBySubscription(context.TODO(), subType, page, pageSize)
		if err != nil {
			return err
		}

		return c.JSON(users)
	})

	app.Post("/users/promote-to-admin", authMiddleware.RequireAdmin(), func(c *fiber.Ctx) error {
		type Request struct {
			UserID string `json:"userID"`
		}

		var req Request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		err := service.PromoteUserToAdmin(context.TODO(), req.UserID)
		if err != nil {
			return err
		}

		return c.SendStatus(fiber.StatusOK)
	})

	app.Post("/users/update-role", authMiddleware.RequireAdmin(), func(c *fiber.Ctx) error {
		type Request struct {
			UserID string    `json:"userID"`
			Role   user.Role `json:"role"`
		}

		var req Request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		if req.Role != user.RoleAdmin && req.Role != user.RoleLinkoUser {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid role"})
		}

		err := service.UpdateUserRole(context.TODO(), req.UserID, req.Role)
		if err != nil {
			return err
		}

		return c.SendStatus(fiber.StatusOK)
	})

	app.Post("/users/update-subscription", authMiddleware.RequireAdmin(), func(c *fiber.Ctx) error {
		type Request struct {
			UserID           string                `json:"userID"`
			SubscriptionType user.SubscriptionType `json:"subscriptionType"`
		}

		var req Request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		if req.SubscriptionType != user.FreeTier &&
			req.SubscriptionType != user.LinkoPlus &&
			req.SubscriptionType != user.LinkoVIP {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid subscription type"})
		}

		err := service.UpdateUserSubscription(context.TODO(), req.UserID, req.SubscriptionType)
		if err != nil {
			return err
		}

		return c.SendStatus(fiber.StatusOK)
	})

	app.Delete("/users/:id", authMiddleware.RequireAdmin(), func(c *fiber.Ctx) error {
		userID := c.Params("id")

		err := service.DeleteUser(context.TODO(), userID)
		if err != nil {
			return err
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	app.Get("/users/whitelist", authMiddleware.RequireAdmin(), func(c *fiber.Ctx) error {
		whitelist, err := service.GetWhitelist(context.TODO())
		if err != nil {
			return err
		}

		return c.JSON(whitelist)
	})

	app.Post("/users/whitelist", authMiddleware.RequireAdmin(), func(c *fiber.Ctx) error {
		type Request struct {
			Email string `json:"email"`
		}

		var req Request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		err := service.AddToWhitelist(context.TODO(), req.Email)
		if err != nil {
			return err
		}

		return c.SendStatus(fiber.StatusOK)
	})

	app.Delete("/users/whitelist", authMiddleware.RequireAdmin(), func(c *fiber.Ctx) error {
		type Request struct {
			Email string `json:"email"`
		}

		var req Request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		err := service.RemoveFromWhitelist(context.TODO(), req.Email)
		if err != nil {
			return err
		}

		return c.SendStatus(fiber.StatusOK)
	})
}
