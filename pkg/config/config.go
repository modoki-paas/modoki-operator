package config

import (
	"context"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/env"
	"github.com/heetch/confita/backend/file"
	"golang.org/x/xerrors"
)

// GitHubPrivateKey represents a private key
type GitHubPrivateKey struct {
	Path string `config:"private_key_path" json:"path" yaml:"path"`
	Raw  string `config:"private_key" json:"raw" yaml:"raw"`
}

// GitHub is a config for GitHub App
type GitHub struct {
	AppID      int64            `json:"appID" yaml:"appID" config:"app-id"`
	PrivateKey GitHubPrivateKey `json:"private_key" yaml:"private_key"`
}

type Config struct {
	GitHub *GitHub `json:"github" yaml:"github"`
}

// ReadConfig reads config from env, json and yaml
func ReadConfig(path string) (*Config, error) {
	loader := confita.NewLoader(
		env.NewBackend(),
		file.NewOptionalBackend(path),
		file.NewOptionalBackend("./modoki.json"),
		file.NewOptionalBackend("./modoki.yaml"),
		file.NewOptionalBackend("/etc/modoki/modoki.json"),
		file.NewOptionalBackend("/etc/modoki/modoki.yaml"),
	)

	cfg := &Config{}

	err := loader.Load(context.Background(), cfg)

	if err != nil {
		return nil, xerrors.Errorf("failed to load config: %w", err)
	}

	return cfg, nil
}
