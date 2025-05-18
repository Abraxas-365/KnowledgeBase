package chatuserapi

import (
	"github.com/Abraxas-365/opd/internal/chatuser"
	"github.com/Abraxas-365/opd/internal/chatuser/chatusersrv"
	"github.com/Abraxas-365/opd/pkg/middleware"
	"github.com/gofiber/fiber/v2"
)

// SetupRoutes sets up the API routes for the chat user service
// Updated to accept JWTAuthMiddleware instead of lucia.AuthMiddleware
func SetupRoutes(
	app *fiber.App,
	service *chatusersrv.Service,
	authMiddleware *middleware.JWTAuthMiddleware,
) {

	// Get chat user by ID
	app.Get("/chat-users/:id", authMiddleware.RequireAuth(), func(c *fiber.Ctx) error {
		chatUserID := c.Params("id")

		user, err := service.GetChatUserByID(c.Context(), chatUserID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch chat user",
			})
		}

		return c.JSON(user)
	})

	// Create new chat user
	app.Post("/chat-users", func(c *fiber.Ctx) error {
		// Check for authorization header
		authHeader := c.Get("Authorization")
		if authHeader != "TsisNotARealLenguage" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authorization",
			})
		}

		var newUser chatuser.ChatUser
		if err := c.BodyParser(&newUser); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		// Validate required fields
		if newUser.Gender == "" || newUser.Ocupation == "" || newUser.Location == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Gender, Occupation and Location are required",
			})
		}

		// Validate age
		if newUser.Age <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Age must be greater than 0",
			})
		}

		createdUser, err := service.CreateChatUser(c.Context(), newUser)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create chat user",
			})
		}

		return c.Status(fiber.StatusCreated).JSON(createdUser)
	})
}
