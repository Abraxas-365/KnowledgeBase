package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Abraxas-365/opd/internal/analitics/analiticsapi"
	analyticsinfra "github.com/Abraxas-365/opd/internal/analitics/analiticsinfra"
	"github.com/Abraxas-365/opd/internal/analitics/analiticssrv"
	"github.com/Abraxas-365/opd/internal/chatuser/chatuserapi"
	"github.com/Abraxas-365/opd/internal/chatuser/chatuserinfra"
	"github.com/Abraxas-365/opd/internal/chatuser/chatusersrv"
	"github.com/Abraxas-365/opd/internal/interaction/interactioninfra"
	"github.com/Abraxas-365/opd/internal/interaction/interactionsrv"
	"github.com/Abraxas-365/opd/internal/kb/kbapi"
	"github.com/Abraxas-365/opd/internal/kb/kbasesrv"
	"github.com/Abraxas-365/opd/internal/kb/kbinfra"
	"github.com/Abraxas-365/opd/internal/user"
	"github.com/Abraxas-365/opd/internal/user/userapi"
	"github.com/Abraxas-365/opd/internal/user/userinfra"
	"github.com/Abraxas-365/opd/internal/user/usersrv"
	"github.com/Abraxas-365/opd/pkg/conf"
	"github.com/Abraxas-365/opd/pkg/jwt"
	"github.com/Abraxas-365/opd/pkg/middleware"
	"github.com/Abraxas-365/toolkit/pkg/errors"
	"github.com/Abraxas-365/toolkit/pkg/lucia"
	"github.com/Abraxas-365/toolkit/pkg/lucia/luciastore"
	"github.com/Abraxas-365/toolkit/pkg/s3client"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/bedrockagent"
	"github.com/gofiber/fiber/v2"
	fiberCors "github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/jmoiron/sqlx"
)

func main() {
	conf := conf.Load()

	db, err := sqlx.Connect("postgres", conf.DatabaseURL)
	if err != nil {
		panic(err)
	}

	// Initialize repositories and services
	userRepo := userinfra.NewUserStore(db)
	userSrv := usersrv.NewService(userRepo)
	analrepo := analyticsinfra.NewAnalyticsStore(db)
	sessionStore := luciastore.NewStoreFromConnection(db)
	authSrv := lucia.NewAuthService[*user.User](userSrv, sessionStore)
	chatUserRepo := chatuserinfra.NewChatUserStore(db)
	chatUserSrv := chatusersrv.New(chatUserRepo)

	s3client, err := s3client.NewS3Client("vendy", s3client.WithRegion("us-east-1"))
	if err != nil {
		panic(err)
	}
	analSrv := analiticssrv.NewService(analrepo, s3client)

	interactionRepo := interactioninfra.NewInteractionStore(db)
	interactionSrv := interactionsrv.New(interactionRepo)

	// Initialize Google OAuth provider
	googleProvider := lucia.NewGoogleProvider(
		conf.GoogleClientID,
		conf.GoogleClientSecret,
		conf.GoogleRedirectURI,
		[]string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
	)
	authSrv.RegisterProvider("google", googleProvider)

	// Initialize AWS services
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
	)
	if err != nil {
		panic("unable to load SDK config: " + err.Error())
	}

	client := bedrockagentruntime.NewFromConfig(cfg)

	repo := kbinfra.NewStore(db)

	brClient := bedrockagent.New(session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	})))

	// Initialize knowledge base service
	kbSerive := kbsrv.New(client, brClient, repo, s3client, *userSrv, *chatUserSrv, *interactionSrv)

	// Initialize JWT service
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key-change-in-production" // For development only
	}

	accessTokenDuration := 1 * time.Hour       // Default 1 hour
	refreshTokenDuration := 7 * 24 * time.Hour // Default 7 days

	// Allow configuring token durations via environment variables
	if durationStr := os.Getenv("JWT_ACCESS_DURATION"); durationStr != "" {
		if seconds, err := strconv.Atoi(durationStr); err == nil && seconds > 0 {
			accessTokenDuration = time.Duration(seconds) * time.Second
		}
	}

	if durationStr := os.Getenv("JWT_REFRESH_DURATION"); durationStr != "" {
		if seconds, err := strconv.Atoi(durationStr); err == nil && seconds > 0 {
			refreshTokenDuration = time.Duration(seconds) * time.Second
		}
	}

	jwtService := jwt.NewJWTService(jwtSecret, accessTokenDuration, refreshTokenDuration)
	jwtMiddleware := middleware.NewJWTAuthMiddleware(jwtService, userSrv)

	app := fiber.New()

	// Apply JWT middleware globally
	app.Use(jwtMiddleware.AuthMiddleware())

	// Add CORS middleware
	app.Use(fiberCors.New(fiberCors.Config{
		AllowOrigins:     conf.AllowOrigins,
		AllowCredentials: true,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
	}))

	// Setup API routes
	kbapi.SetupRoutes(app, kbSerive, jwtMiddleware)
	userapi.SetupRoutes(app, userSrv, jwtMiddleware)
	analiticsapi.SetupRoutes(app, analSrv, jwtMiddleware)
	chatuserapi.SetupRoutes(app, chatUserSrv, jwtMiddleware)

	// Google OAuth routes
	app.Get("/login/google", func(c *fiber.Ctx) error {
		authURL, state, err := authSrv.GetAuthURL("google")
		if err != nil {
			return err
		}
		c.Cookie(&fiber.Cookie{
			Name:     "oauth_state",
			Value:    state,
			HTTPOnly: true,
			Secure:   true,
		})
		return c.Redirect(authURL)
	})

	app.Get("/login/google/callback", func(c *fiber.Ctx) error {
		state := c.Cookies("oauth_state")
		if state == "" || state != c.Query("state") {
			return errors.ErrUnauthorized("Invalid state")
		}

		code := c.Query("code")
		if code == "" {
			return errors.ErrBadRequest("Missing code")
		}

		// Use the lucia auth service to handle the OAuth callback
		session, err := authSrv.HandleCallback(c.Context(), "google", code)
		if err != nil {
			return err
		}

		// Get user from session
		userID, err := session.UserIDToString()
		if err != nil {
			return err
		}

		user, err := userSrv.GetUser(c.Context(), userID)
		if err != nil {
			return err
		}

		// Generate JWT tokens
		accessToken, err := jwtService.GenerateToken(user)
		if err != nil {
			return errors.ErrUnexpected("Failed to generate access token")
		}

		refreshToken, err := jwtService.GenerateRefreshToken(user.ID)
		if err != nil {
			return errors.ErrUnexpected("Failed to generate refresh token")
		}

		// Return JSON with tokens instead of using cookies
		return c.JSON(fiber.Map{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"expires_in":    int(accessTokenDuration.Seconds()),
			"token_type":    "Bearer",
			"user":          user,
		})
	})

	// Refresh token endpoint
	app.Post("/refresh-token", func(c *fiber.Ctx) error {
		type Request struct {
			RefreshToken string `json:"refresh_token"`
		}

		var req Request
		if err := c.BodyParser(&req); err != nil {
			return errors.ErrBadRequest("Invalid request body")
		}

		// Validate refresh token
		userID, err := jwtService.ValidateRefreshToken(req.RefreshToken)
		if err != nil {
			return errors.ErrUnauthorized("Invalid refresh token")
		}

		// Get user
		user, err := userSrv.GetUser(c.Context(), userID)
		if err != nil {
			return err
		}

		// Generate new access token
		accessToken, err := jwtService.GenerateToken(user)
		if err != nil {
			return errors.ErrUnexpected("Failed to generate access token")
		}

		return c.JSON(fiber.Map{
			"access_token": accessToken,
			"expires_in":   int(accessTokenDuration.Seconds()),
			"token_type":   "Bearer",
		})
	})

	// Logout is a client-side operation with JWT, but we keep an endpoint for compatibility
	app.Post("/logout", func(c *fiber.Ctx) error {
		// JWT logout is handled client-side by discarding the tokens
		return c.SendString("Logged out successfully")
	})

	// Start server
	fmt.Printf("Starting server on port %s\n", conf.Port)
	app.Listen(conf.Port)
}
