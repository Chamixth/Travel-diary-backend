package middleware

import (
	"errors"
	"log"
	"strings"

	"travel-diary-backend/internal/config"
	"travel-diary-backend/internal/dao"
	"travel-diary-backend/internal/integrations"
	"travel-diary-backend/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
)

const auth0UserIDKey = "auth0UserID"

func Auth0(cfg config.Config, db *mongo.Database) fiber.Handler {
	users := dao.NewUserDAO(db)

	return func(c *fiber.Ctx) error {
		token := c.Get("Authorization")
		if token == "" {
			log.Printf("auth0: missing Authorization header: %s %s", c.Method(), c.Path())
			return fiber.ErrUnauthorized
		}
		if len(token) < 7 || token[:7] != "Bearer " {
			log.Printf("auth0: malformed Authorization header: %s %s", c.Method(), c.Path())
			return fiber.ErrUnauthorized
		}
		raw := token[7:]
		if raw == "" {
			log.Printf("auth0: empty bearer token: %s %s", c.Method(), c.Path())
			return fiber.ErrUnauthorized
		}

		jwksURL := cfg.Auth0JWKSURL
		if jwksURL == "" && cfg.Auth0Domain != "" {
			jwksURL = "https://" + cfg.Auth0Domain + "/.well-known/jwks.json"
		}
		if jwksURL == "" || cfg.Auth0Issuer == "" || cfg.Auth0Audience == "" {
			log.Printf("auth0: incomplete configuration (domain=%t issuer=%t audience=%t jwks=%t)", cfg.Auth0Domain != "", cfg.Auth0Issuer != "", cfg.Auth0Audience != "", jwksURL != "")
			return fiber.ErrInternalServerError
		}

		_, claims, err := integrations.ParseAuth0Token(c.Context(), jwksURL, cfg.Auth0Issuer, cfg.Auth0Audience, raw)
		if err != nil {
			log.Printf("auth0: token validation failed for %s %s: %v (expected issuer=%q audience=%q)", c.Method(), c.Path(), err, cfg.Auth0Issuer, cfg.Auth0Audience)
			return fiber.ErrUnauthorized
		}

		sub, _ := claims["sub"].(string)
		if sub == "" {
			log.Printf("auth0: validated token has no sub claim: %s %s", c.Method(), c.Path())
			return fiber.ErrUnauthorized
		}

		user, err := users.FindByAuth0ID(c.Context(), sub)
		if errors.Is(err, dao.ErrUserNotFound) {
			user, err = users.UpsertAuth0User(c.Context(), models.User{
				ID:           uuid.NewString(),
				Auth0ID:      sub,
				AuthProvider: "auth0",
				Username:     auth0Username(claims, sub),
				Email:        auth0Claim(claims, "email"),
				Name:         auth0Name(claims),
				PictureURL:   auth0Claim(claims, "picture"),
			})
		}
		if err != nil {
			log.Printf("auth0: user provisioning failed for sub %q: %v", sub, err)
			return fiber.ErrUnauthorized
		}

		c.Locals(auth0UserIDKey, user.ID)
		c.Locals("auth0Sub", sub)
		c.Locals("user", user)
		return c.Next()
	}
}

func auth0Claim(claims map[string]any, key string) string {
	value, _ := claims[key].(string)
	return strings.TrimSpace(value)
}

func auth0Name(claims map[string]any) string {
	name := auth0Claim(claims, "name")
	if name != "" {
		return name
	}
	return auth0Claim(claims, "nickname")
}

func auth0Username(claims map[string]any, sub string) string {
	username := auth0Claim(claims, "nickname")
	if username == "" {
		username = auth0Claim(claims, "email")
		if at := strings.IndexByte(username, '@'); at > 0 {
			username = username[:at]
		}
	}
	if username == "" {
		username = strings.ReplaceAll(sub, "|", "-")
	}
	return username
}

func Auth0UserID(c *fiber.Ctx) (string, bool) {
	userID, ok := c.Locals(auth0UserIDKey).(string)
	return userID, ok && userID != ""
}
