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
	Addr       string        `yaml:"addr"`
	SigningKey string        `yaml:"signingKey"`
	TokenTTL   time.Duration `yaml:"tokenTTL"`
	BaseURL    string        `yaml:"baseURL"`
}

func Load() *Config {
	// 默认值
	cfg := &Config{
		Addr:       ":8080",
		SigningKey: "change-me",
		TokenTTL:   3 * time.Hour,
		BaseURL:    "",
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

	// flags 覆盖
	addr := flag.String("addr", cfg.Addr, "server listen address")
	signingKey := flag.String("signing-key", cfg.SigningKey, "HMAC signing key")
	tokenTTL := flag.Duration("token-ttl", cfg.TokenTTL, "signed link ttl")
	baseURL := flag.String("base-url", cfg.BaseURL, "public base URL, e.g. https://dl.example.com")
	flag.Parse()

	cfg.Addr = *addr
	cfg.SigningKey = *signingKey
	cfg.TokenTTL = *tokenTTL
	cfg.BaseURL = *baseURL

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
