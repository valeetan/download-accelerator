package config

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Addr          string        `yaml:"addr"`
	SigningKey    string        `yaml:"signingKey"`
	TokenTTL      time.Duration `yaml:"tokenTTL"`
	BaseURL       string        `yaml:"baseURL"`
	DBPath        string        `yaml:"dbPath"`
	HTTPTimeout   time.Duration `yaml:"httpTimeout"`
	MaxConcurrent int           `yaml:"maxConcurrent"`
	AdminUser     string        `yaml:"adminUser"`
	AdminPass     string        `yaml:"adminPass"`
	ThrottleBps   int           `yaml:"throttleBps"`
	// 指纹限制策略
	FingerprintLimitPerHour   int  `yaml:"fingerprintLimitPerHour"`   // 每指纹每小时生成限制
	RestrictToSameFingerprint bool `yaml:"restrictToSameFingerprint"` // 是否仅允许同指纹下载
}

func Load() *Config {
	// 默认值
	cfg := &Config{
		Addr:                      ":8080",
		SigningKey:                "change-me",
		TokenTTL:                  3 * time.Hour,
		BaseURL:                   "",
		DBPath:                    "data/app.db",
		HTTPTimeout:               15 * time.Second,
		MaxConcurrent:             16,
		AdminUser:                 "admin",
		AdminPass:                 "",
		ThrottleBps:               0,
		FingerprintLimitPerHour:   10,    // 默认每指纹每小时最多生成10个链接
		RestrictToSameFingerprint: false, // 默认不限制下载指纹
	}

	// 从 YAML 读取（可选）
	if path, ok := findConfigFile(); ok {
		if b, err := os.ReadFile(path); err == nil {
			_ = yaml.Unmarshal(b, cfg)
		}
	}

	// env 覆盖
	if v := os.Getenv("APP_ADDR"); v != "" {
		cfg.Addr = v
	}
	if v := os.Getenv("APP_SIGNING_KEY"); v != "" {
		cfg.SigningKey = v
	}
	if v := os.Getenv("APP_TOKEN_TTL"); v != "" {
		cfg.TokenTTL = parseDuration(v, cfg.TokenTTL)
	}
	if v := os.Getenv("APP_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}
	if v := os.Getenv("APP_DB_PATH"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("APP_HTTP_TIMEOUT"); v != "" {
		cfg.HTTPTimeout = parseDuration(v, cfg.HTTPTimeout)
	}
	if v := os.Getenv("APP_MAX_CONCURRENT"); v != "" {
		cfg.MaxConcurrent = parseInt(v, cfg.MaxConcurrent)
	}
	if v := os.Getenv("APP_ADMIN_USER"); v != "" {
		cfg.AdminUser = v
	}
	if v := os.Getenv("APP_ADMIN_PASS"); v != "" {
		cfg.AdminPass = v
	}
	if v := os.Getenv("APP_THROTTLE_BPS"); v != "" {
		cfg.ThrottleBps = parseInt(v, cfg.ThrottleBps)
	}

	// flags 覆盖
	addr := flag.String("addr", cfg.Addr, "server listen address")
	signingKey := flag.String("signing-key", cfg.SigningKey, "HMAC signing key")
	tokenTTL := flag.Duration("token-ttl", cfg.TokenTTL, "signed link ttl")
	baseURL := flag.String("base-url", cfg.BaseURL, "public base URL, e.g. https://dl.example.com")
	dbPath := flag.String("db", cfg.DBPath, "sqlite db path")
	httpTimeout := flag.Duration("http-timeout", cfg.HTTPTimeout, "http client timeout")
	maxConc := flag.Int("max-concurrent", cfg.MaxConcurrent, "max concurrent proxy workers")
	adminUser := flag.String("admin-user", cfg.AdminUser, "bootstrap admin username")
	adminPass := flag.String("admin-pass", cfg.AdminPass, "bootstrap admin password (used once)")
	throttle := flag.Int("throttle-bps", cfg.ThrottleBps, "proxy throttle bytes/sec (0=unlimited)")
	flag.Parse()

	cfg.Addr = *addr
	cfg.SigningKey = *signingKey
	cfg.TokenTTL = *tokenTTL
	cfg.BaseURL = *baseURL
	cfg.DBPath = *dbPath
	cfg.HTTPTimeout = *httpTimeout
	cfg.MaxConcurrent = *maxConc
	cfg.AdminUser = *adminUser
	cfg.AdminPass = *adminPass
	cfg.ThrottleBps = *throttle

	if cfg.SigningKey == "change-me" {
		log.Println("WARNING: APP_SIGNING_KEY 未设置为安全值，请在生产环境修改")
	}

	return cfg
}

func parseDuration(s string, def time.Duration) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}
	return d
}

func parseInt(s string, def int) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			continue
		}
		n = n*10 + int(s[i]-'0')
	}
	if n == 0 {
		return def
	}
	return n
}

func findConfigFile() (string, bool) {
	// 优先 configs/config.yaml
	paths := []string{
		"configs/config.yaml",
		"config.yaml",
	}
	for _, p := range paths {
		if exists(p) {
			return p, true
		}
	}
	return "", false
}

func exists(p string) bool {
	if p == "" {
		return false
	}
	if info, err := os.Stat(filepath.Clean(p)); err == nil {
		return !info.IsDir()
	}
	return false
}
