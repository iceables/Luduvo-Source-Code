package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

//handle and provide configurations

type Config struct {
	DBUser                 string
	DBPass                 string
	DBHost                 string
	DBPort                 string
	DBName                 string
	RunWithDatabase        bool
	DBMaxOpenConns         int
	DBMaxIdleConns         int
	DBConnMaxLifetime      time.Duration
	DBConnMaxIdleTime      time.Duration
	ServerReadTimeout      time.Duration
	ServerWriteTimeout     time.Duration
	ServerIdleTimeout      time.Duration
	LimiterMax             int
	LimiterExpiration      time.Duration
	SessionSecure          bool
	SessionSameSite        string
	SessionIdleTimeout     time.Duration
	SessionAbsoluteTimeout time.Duration
	AltchaHMACKey          string
	GameserverAPIKey       string
	LiveTimerEnabled       bool
	LiveTimerDuration      time.Duration
	LiveTimerEnd           time.Time
}

var Global *Config

func parseLiveTimerDuration(val string) time.Duration {
	val = strings.TrimSpace(val)
	if val == "" {
		return 0
	}
	if d, err := time.ParseDuration(val); err == nil && d > 0 {
		return d
	}
	if sec, err := strconv.Atoi(val); err == nil && sec > 0 {
		return time.Duration(sec) * time.Second
	}
	if strings.Contains(val, ":") {
		parts := strings.Split(val, ":")
		if len(parts) == 2 {
			m, err1 := strconv.Atoi(parts[0])
			s, err2 := strconv.Atoi(parts[1])
			if err1 == nil && err2 == nil && m >= 0 && s >= 0 {
				return time.Duration(m)*time.Minute + time.Duration(s)*time.Second
			}
		}
	}
	return 0
}

func (c *Config) IsLiveTimerActive() bool {
	return c != nil && c.LiveTimerEnabled && c.LiveTimerDuration > 0 && time.Now().Before(c.LiveTimerEnd)
}

func (c *Config) LiveTimerRemaining() time.Duration {
	if !c.IsLiveTimerActive() {
		return 0
	}
	rem := time.Until(c.LiveTimerEnd)
	if rem < 0 {
		return 0
	}
	return rem
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	runWithDB, _ := strconv.ParseBool(os.Getenv("RUN_WITH_DATABASE"))

	dbMaxOpenConns, _ := strconv.Atoi(os.Getenv("DB_MAX_OPEN_CONNS"))
	dbMaxIdleConns, _ := strconv.Atoi(os.Getenv("DB_MAX_IDLE_CONNS"))
	dbConnMaxLifetime, _ := time.ParseDuration(os.Getenv("DB_CONN_MAX_LIFETIME"))
	dbConnMaxIdleTime, _ := time.ParseDuration(os.Getenv("DB_CONN_MAX_IDLE_TIME"))
	serverReadTimeout, _ := time.ParseDuration(os.Getenv("SERVER_READ_TIMEOUT"))
	serverWriteTimeout, _ := time.ParseDuration(os.Getenv("SERVER_WRITE_TIMEOUT"))
	serverIdleTimeout, _ := time.ParseDuration(os.Getenv("SERVER_IDLE_TIMEOUT"))
	limiterMax, _ := strconv.Atoi(os.Getenv("LIMITER_MAX"))
	limiterExpiration, _ := time.ParseDuration(os.Getenv("LIMITER_EXPIRATION"))
	sessionSecure, _ := strconv.ParseBool(os.Getenv("SESSION_SECURE"))
	sessionSameSite := os.Getenv("SESSION_SAMESITE")
	sessionIdleTimeout, _ := time.ParseDuration(os.Getenv("SESSION_IDLE_TIMEOUT"))
	sessionAbsoluteTimeout, _ := time.ParseDuration(os.Getenv("SESSION_ABSOLUTE_TIMEOUT"))
	altchaHMACKey := os.Getenv("ALTCHA_HMAC_KEY")

	liveTimerEnabled, _ := strconv.ParseBool(os.Getenv("LIVETIMER_ENABLED"))
	liveTimerDuration := parseLiveTimerDuration(os.Getenv("LIVETIMER_DURATION"))
	if liveTimerDuration > time.Hour {
		liveTimerDuration = time.Hour
	}
	liveTimerEnd := time.Now().Add(liveTimerDuration)

	cfg := &Config{
		DBUser:                 os.Getenv("DB_USER"),
		DBPass:                 os.Getenv("DB_PASS"),
		DBHost:                 os.Getenv("DB_HOST"),
		DBPort:                 os.Getenv("DB_PORT"),
		DBName:                 os.Getenv("DB_NAME"),
		RunWithDatabase:        runWithDB, //testing purposes
		DBMaxOpenConns:         dbMaxOpenConns,
		DBMaxIdleConns:         dbMaxIdleConns,
		DBConnMaxLifetime:      dbConnMaxLifetime,
		DBConnMaxIdleTime:      dbConnMaxIdleTime,
		ServerReadTimeout:      serverReadTimeout,
		ServerWriteTimeout:     serverWriteTimeout,
		ServerIdleTimeout:      serverIdleTimeout,
		LimiterMax:             limiterMax,
		LimiterExpiration:      limiterExpiration,
		SessionSecure:          sessionSecure,
		SessionSameSite:        sessionSameSite,
		SessionIdleTimeout:     sessionIdleTimeout,
		SessionAbsoluteTimeout: sessionAbsoluteTimeout,
		AltchaHMACKey:          altchaHMACKey,
		GameserverAPIKey:       os.Getenv("GAMESERVER_API_KEY"),
		LiveTimerEnabled:       liveTimerEnabled,
		LiveTimerDuration:      liveTimerDuration,
		LiveTimerEnd:           liveTimerEnd,
	}

	Global = cfg

	//return to database.go the config
	return cfg, nil
}