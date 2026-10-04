package config

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Config struct {
	AppName                 string
	Environment             string
	Port                    string
	DatabaseURL             string
	MongoDatabase           string
	AWSRegion               string
	AWSAccessKeyID          string
	AWSSecretAccessKey      string
	AWSBucket               string
	JWTSecret               string
	Auth0Domain             string
	Auth0Audience           string
	Auth0Issuer             string
	Auth0JWKSURL            string
	CORSAllowedOrigins      string
	GoogleClientID          string
	GoogleClientSecret      string
	GoogleAuthStateSecret   string
	GoogleRedirectURL       string
	FrontendRedirectURL     string
	OpenAIKey               string
	OpenAIModelImage        string
	OpenAIModelCreative     string
	CreativePipelineEnabled bool
	ImageAnalysisWorkers    int
	GenerationWorkers       int
	GenerationMaxAttempts   int
}

func Load() Config {
	loadDotEnv(".env")

	return Config{
		AppName:                 getEnv("APP_NAME", "travel-diary-backend"),
		Environment:             getEnv("APP_ENV", "development"),
		Port:                    getEnv("PORT", "8001"),
		DatabaseURL:             normalizeDatabaseURL(getEnv("DATABASE_URL", "")),
		MongoDatabase:           getEnv("MONGO_DATABASE", "travel_diary"),
		AWSRegion:               getEnv("AWS_REGION", ""),
		AWSAccessKeyID:          getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretAccessKey:      getEnv("AWS_SECRET_ACCESS_KEY", ""),
		AWSBucket:               getEnv("AWS_S3_BUCKET", ""),
		JWTSecret:               getEnv("JWT_SECRET", "change-me"),
		Auth0Domain:             getEnv("AUTH0_DOMAIN", ""),
		Auth0Audience:           getEnv("AUTH0_AUDIENCE", ""),
		Auth0Issuer:             getEnv("AUTH0_ISSUER", ""),
		Auth0JWKSURL:            getEnv("AUTH0_JWKS_URL", ""),
		CORSAllowedOrigins:      sanitizeOrigins(getEnv("CORS_ALLOWED_ORIGINS", "")),
		GoogleClientID:          getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:      getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleAuthStateSecret:   getEnv("GOOGLE_AUTH_STATE_SECRET", "state-secret"),
		GoogleRedirectURL:       getEnv("GOOGLE_REDIRECT_URL", ""),
		FrontendRedirectURL:     getEnv("FRONTEND_REDIRECT_URL", ""),
		OpenAIKey:               getEnv("OPENAI_KEY", ""),
		OpenAIModelImage:        getEnv("OPENAI_MODEL_IMAGE_ANALYSIS", "gpt-4o-mini"),
		OpenAIModelCreative:     getEnv("OPENAI_MODEL_CREATIVE", getEnv("OPENAI_MODEL_IMAGE_ANALYSIS", "gpt-4o-mini")),
		CreativePipelineEnabled: getEnvBool("CREATIVE_PIPELINE_ENABLED", false),
		ImageAnalysisWorkers:    getEnvInt("IMAGE_ANALYSIS_WORKERS", 4),
		GenerationWorkers:       getEnvInt("GENERATION_WORKERS", 1),
		GenerationMaxAttempts:   getEnvInt("GENERATION_MAX_ATTEMPTS", 3),
	}
}

func normalizeDatabaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, `"'`)
	raw = strings.TrimPrefix(raw, "export DATABASE_URL=")
	raw = strings.TrimPrefix(raw, "DATABASE_URL=")
	raw = strings.TrimPrefix(raw, "[")
	raw = strings.TrimSuffix(raw, "]")
	return raw
}

func loadDotEnv(path string) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = sanitizeEnvValue(strings.TrimSpace(value))
		if key == "" || value == "" {
			continue
		}
		// A real environment variable always wins over the .env file, so a
		// deployment that injects config through the environment (Railway,
		// Render, ...) is never shadowed by a stray committed .env.
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, value)
	}
}

var markdownLinkPattern = regexp.MustCompile(`^\[([^\]]+)\]\([^)]+\)$`)

func sanitizeEnvValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	if match := markdownLinkPattern.FindStringSubmatch(value); len(match) == 2 {
		return match[1]
	}
	return value
}

// sanitizeOrigins turns a loosely formatted CORS origins env value into a
// clean comma-separated list. It tolerates commas, newlines, spaces, tabs and
// surrounding quotes/brackets as separators — dashboards (Vercel especially)
// often store the value newline-separated, which makes Fiber's CORS
// middleware panic with "Invalid origin format".
func sanitizeOrigins(raw string) string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	})

	clean := make([]string, 0, len(fields))
	seen := make(map[string]bool)

	for _, part := range fields {
		part = sanitizeEnvValue(part)
		if part == "" {
			continue
		}

		u, err := url.Parse(part)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			continue
		}
		// An origin is scheme://host[:port] only — no path, query or fragment.
		if u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
			continue
		}

		origin := u.Scheme + "://" + u.Host
		if !seen[origin] {
			seen[origin] = true
			clean = append(clean, origin)
		}
	}

	return strings.Join(clean, ",")
}

func (c Config) ServerAddress() string {
	return ":" + c.Port
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	var parsed int
	if _, err := fmt.Sscanf(value, "%d", &parsed); err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func ValidateDatabaseURL(raw string) error {
	if raw == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if parsed.Scheme != "mongodb" && parsed.Scheme != "mongodb+srv" {
		return fmt.Errorf("DATABASE_URL must start with mongodb:// or mongodb+srv://")
	}
	return nil
}
