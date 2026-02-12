package config

import (
	"os"
)

type Config struct {
	AppName             string
	Version             string
	FrontendURL         string
	IssuerURL           string
	IssuerInternalURL   string
	EmailSenderURL      string
	DiscordClientID     string
	DiscordClientSecret string
	GitHubClientID      string
	GitHubClientSecret  string
}

// envが設定されていない場合のデフォルト値
var (
	Version   = "latest"
	GitCommit = "unknown"
	GitBranch = "unknown"
)

var (
	AppName           = "UniQUE"
	FrontendURL       = "http://localhost:3000"
	IssuerURL         = "http://localhost:8080"
	IssuerInternalURL = "http://localhost:8080"
	EmailSenderURL    = "http://localhost:8080"
)

func LoadConfig() *Config {
	version := Version

	if Version == "latest" {
		version = GitBranch + "@" + GitCommit
	} else {
		version = Version + "+" + GitCommit
	}

	// envから設定を読み込む
	AppNameEnv := os.Getenv("CONFIG_APP_NAME")
	if AppNameEnv == "" {
		AppNameEnv = AppName
	}
	FrontendURLEnv := os.Getenv("CONFIG_FRONTEND_URL")
	if FrontendURLEnv == "" {
		FrontendURLEnv = FrontendURL
	}
	IssuerURLEnv := os.Getenv("CONFIG_ISSUER_URL")
	if IssuerURLEnv == "" {
		IssuerURLEnv = IssuerURL
	}
	IssuerInternalURLEnv := os.Getenv("CONFIG_ISSUER_INTERNAL_URL")
	if IssuerInternalURLEnv == "" {
		IssuerInternalURLEnv = IssuerInternalURL
	}
	EmailSenderURLEnv := os.Getenv("CONFIG_EMAIL_SENDER_URL")
	if EmailSenderURLEnv == "" {
		EmailSenderURLEnv = EmailSenderURL
	}
	return &Config{
		AppName:             AppNameEnv,
		FrontendURL:         FrontendURLEnv,
		IssuerURL:           IssuerURLEnv,
		IssuerInternalURL:   IssuerInternalURLEnv,
		EmailSenderURL:      EmailSenderURLEnv,
		Version:             version,
		DiscordClientID:     os.Getenv("DISCORD_CLIENT_ID"),
		DiscordClientSecret: os.Getenv("DISCORD_CLIENT_SECRET"),
		GitHubClientID:      os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret:  os.Getenv("GITHUB_CLIENT_SECRET"),
	}
}
