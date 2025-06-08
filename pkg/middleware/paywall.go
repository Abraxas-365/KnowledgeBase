package middleware

import (
	"sync"
	"time"

	"github.com/Abraxas-365/opd/internal/user"
	"github.com/Abraxas-365/toolkit/pkg/errors"
	"github.com/gofiber/fiber/v2"
)

// RateLimitConfig defines the configuration for rate limiting
type RateLimitConfig struct {
	// Non-authenticated users (by IP)
	AnonymousLimit  int           // 5 requests
	AnonymousWindow time.Duration // time window

	// Authenticated users (by user ID)
	FreeTierLimit  int // e.g., 50 requests
	FreeTierWindow time.Duration

	LinkoPlusLimit  int // e.g., 200 requests
	LinkoPlusWindow time.Duration

	LinkoVIPLimit  int // e.g., 1000 requests
	LinkoVIPWindow time.Duration

	// Admins have unlimited access (no rate limiting)

	// Cleanup interval for expired entries
	CleanupInterval time.Duration
}

// DefaultRateLimitConfig provides sensible defaults
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		AnonymousLimit:  5,
		AnonymousWindow: 24 * time.Hour, // 5 requests per day for anonymous users

		FreeTierLimit:  50,
		FreeTierWindow: 24 * time.Hour, // 50 requests per day for free tier

		LinkoPlusLimit:  200,
		LinkoPlusWindow: 24 * time.Hour, // 200 requests per day for plus

		LinkoVIPLimit:  1000,
		LinkoVIPWindow: 24 * time.Hour, // 1000 requests per day for VIP

		CleanupInterval: 1 * time.Hour, // Clean up expired entries every hour
	}
}

// rateLimitEntry stores the rate limit state for a key
type rateLimitEntry struct {
	count     int
	resetTime time.Time
	mutex     sync.RWMutex
}

// PaywallRateLimiter implements sophisticated rate limiting with paywall logic
type PaywallRateLimiter struct {
	config     RateLimitConfig
	ipLimits   map[string]*rateLimitEntry // For anonymous users (IP-based)
	userLimits map[string]*rateLimitEntry // For authenticated users (user ID-based)
	mutex      sync.RWMutex
	stopChan   chan struct{}
}

// NewPaywallRateLimiter creates a new paywall rate limiter
func NewPaywallRateLimiter(config RateLimitConfig) *PaywallRateLimiter {
	limiter := &PaywallRateLimiter{
		config:     config,
		ipLimits:   make(map[string]*rateLimitEntry),
		userLimits: make(map[string]*rateLimitEntry),
		stopChan:   make(chan struct{}),
	}

	// Start cleanup goroutine
	go limiter.cleanup()

	return limiter
}

// cleanup removes expired entries periodically
func (p *PaywallRateLimiter) cleanup() {
	ticker := time.NewTicker(p.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.cleanupExpiredEntries()
		case <-p.stopChan:
			return
		}
	}
}

// cleanupExpiredEntries removes expired rate limit entries
func (p *PaywallRateLimiter) cleanupExpiredEntries() {
	now := time.Now()

	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Clean IP limits
	for key, entry := range p.ipLimits {
		entry.mutex.RLock()
		expired := now.After(entry.resetTime)
		entry.mutex.RUnlock()

		if expired {
			delete(p.ipLimits, key)
		}
	}

	// Clean user limits
	for key, entry := range p.userLimits {
		entry.mutex.RLock()
		expired := now.After(entry.resetTime)
		entry.mutex.RUnlock()

		if expired {
			delete(p.userLimits, key)
		}
	}
}

// Stop stops the cleanup goroutine
func (p *PaywallRateLimiter) Stop() {
	close(p.stopChan)
}

// checkRateLimit checks if a request should be allowed
func (p *PaywallRateLimiter) checkRateLimit(key string, limit int, window time.Duration, isUserBased bool) (bool, int, time.Time, error) {
	now := time.Now()

	var limitsMap map[string]*rateLimitEntry
	if isUserBased {
		limitsMap = p.userLimits
	} else {
		limitsMap = p.ipLimits
	}

	p.mutex.RLock()
	entry, exists := limitsMap[key]
	p.mutex.RUnlock()

	if !exists {
		// Create new entry
		entry = &rateLimitEntry{
			count:     0,
			resetTime: now.Add(window),
		}

		p.mutex.Lock()
		limitsMap[key] = entry
		p.mutex.Unlock()
	}

	entry.mutex.Lock()
	defer entry.mutex.Unlock()

	// Check if window has expired
	if now.After(entry.resetTime) {
		entry.count = 0
		entry.resetTime = now.Add(window)
	}

	// Check if limit is exceeded
	if entry.count >= limit {
		return false, entry.count, entry.resetTime, nil
	}

	// Increment count
	entry.count++

	return true, entry.count, entry.resetTime, nil
}

// RateLimitMiddleware creates a Fiber middleware for paywall rate limiting
func (p *PaywallRateLimiter) RateLimitMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if user is authenticated
		authenticated := c.Locals("authenticated")
		isAuth := authenticated != nil && authenticated.(bool)

		if !isAuth {
			// Anonymous user - rate limit by IP
			ip := c.IP()
			allowed, count, resetTime, err := p.checkRateLimit(
				ip,
				p.config.AnonymousLimit,
				p.config.AnonymousWindow,
				false,
			)

			if err != nil {
				return errors.ErrUnexpected("Rate limiting error")
			}

			// Set rate limit headers
			c.Set("X-RateLimit-Limit", string(rune(p.config.AnonymousLimit)))
			c.Set("X-RateLimit-Remaining", string(rune(p.config.AnonymousLimit-count)))
			c.Set("X-RateLimit-Reset", string(rune(resetTime.Unix())))

			if !allowed {
				return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
					"error":           "Rate limit exceeded for anonymous users. Please register for a free account to get higher limits.",
					"limit":           p.config.AnonymousLimit,
					"reset_time":      resetTime.Unix(),
					"upgrade_message": "Sign up for free to get 50 requests per day, or upgrade to Linko Plus for 200 requests per day!",
				})
			}

			return c.Next()
		}

		// Authenticated user - check role and subscription
		userRole := c.Locals("role")

		// Admins have unlimited access
		if userRole != nil && userRole.(user.Role) == user.RoleAdmin {
			c.Set("X-RateLimit-Limit", "unlimited")
			c.Set("X-RateLimit-Remaining", "unlimited")
			return c.Next()
		}

		// For linko users, check subscription type
		if userRole != nil && userRole.(user.Role) == user.RoleLinkoUser {
			userID := c.Locals("userID").(string)
			subscription := c.Locals("subscriptionType")

			if subscription == nil {
				return errors.ErrForbidden("No subscription found")
			}

			userSub := subscription.(user.SubscriptionType)
			var limit int
			var window time.Duration
			var tierName string

			switch userSub {
			case user.FreeTier:
				limit = p.config.FreeTierLimit
				window = p.config.FreeTierWindow
				tierName = "Free Tier"
			case user.LinkoPlus:
				limit = p.config.LinkoPlusLimit
				window = p.config.LinkoPlusWindow
				tierName = "Linko Plus"
			case user.LinkoVIP:
				limit = p.config.LinkoVIPLimit
				window = p.config.LinkoVIPWindow
				tierName = "Linko VIP"
			default:
				return errors.ErrForbidden("Invalid subscription type")
			}

			allowed, count, resetTime, err := p.checkRateLimit(
				userID,
				limit,
				window,
				true,
			)

			if err != nil {
				return errors.ErrUnexpected("Rate limiting error")
			}

			// Set rate limit headers
			c.Set("X-RateLimit-Limit", string(rune(limit)))
			c.Set("X-RateLimit-Remaining", string(rune(limit-count)))
			c.Set("X-RateLimit-Reset", string(rune(resetTime.Unix())))
			c.Set("X-RateLimit-Tier", tierName)

			if !allowed {
				upgradeMessage := ""
				switch userSub {
				case user.FreeTier:
					upgradeMessage = "Upgrade to Linko Plus for 200 requests per day, or Linko VIP for 1000 requests per day!"
				case user.LinkoPlus:
					upgradeMessage = "Upgrade to Linko VIP for 1000 requests per day!"
				}

				return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
					"error":           "Rate limit exceeded for " + tierName,
					"limit":           limit,
					"reset_time":      resetTime.Unix(),
					"tier":            tierName,
					"upgrade_message": upgradeMessage,
				})
			}

			return c.Next()
		}

		// Fallback for unknown user types
		return errors.ErrForbidden("Unable to determine user access level")
	}
}
