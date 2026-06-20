//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type settingUniFedRepoStub struct {
	values map[string]string
}

func (s *settingUniFedRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	if value, ok := s.values[key]; ok {
		return &Setting{Key: key, Value: value}, nil
	}
	return nil, ErrSettingNotFound
}

func (s *settingUniFedRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}

func (s *settingUniFedRepoStub) Set(ctx context.Context, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func (s *settingUniFedRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *settingUniFedRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *settingUniFedRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *settingUniFedRepoStub) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func TestGetUniFedConnectOAuthConfigUsesDefaultInstanceAndDBOverride(t *testing.T) {
	repo := &settingUniFedRepoStub{values: map[string]string{
		SettingKeyUniFedConnectEnabled:     "true",
		SettingKeyUniFedConnectRedirectURL: "https://app.example.com/api/v1/auth/oauth/unifed/callback",
	}}
	svc := NewSettingService(repo, &config.Config{
		UniFed: config.UniFedConnectConfig{
			Enabled:     false,
			InstanceURL: "",
			RedirectURL: "",
		},
	})

	cfg, err := svc.GetUniFedConnectOAuthConfig(context.Background())

	require.NoError(t, err)
	require.True(t, cfg.Enabled)
	require.Equal(t, "https://dc.hhhl.cc", cfg.InstanceURL)
	require.Equal(t, "https://app.example.com/api/v1/auth/oauth/unifed/callback", cfg.RedirectURL)

	repo.values[SettingKeyUniFedConnectInstanceURL] = "https://misskey.example"
	cfg, err = svc.GetUniFedConnectOAuthConfig(context.Background())

	require.NoError(t, err)
	require.Equal(t, "https://misskey.example", cfg.InstanceURL)
}

func TestGetUniFedConnectOAuthConfigRejectsDisabledAndInvalidURLs(t *testing.T) {
	svc := NewSettingService(&settingUniFedRepoStub{values: map[string]string{
		SettingKeyUniFedConnectEnabled: "false",
	}}, &config.Config{
		UniFed: config.UniFedConnectConfig{
			Enabled:     true,
			InstanceURL: "https://dc.hhhl.cc",
			RedirectURL: "https://app.example.com/api/v1/auth/oauth/unifed/callback",
		},
	})

	_, err := svc.GetUniFedConnectOAuthConfig(context.Background())
	require.Error(t, err)
	require.Equal(t, "OAUTH_DISABLED", infraerrors.Reason(err))

	svc = NewSettingService(&settingUniFedRepoStub{values: map[string]string{
		SettingKeyUniFedConnectEnabled:     "true",
		SettingKeyUniFedConnectInstanceURL: "dc.hhhl.cc",
		SettingKeyUniFedConnectRedirectURL: "https://app.example.com/api/v1/auth/oauth/unifed/callback",
	}}, &config.Config{})

	_, err = svc.GetUniFedConnectOAuthConfig(context.Background())
	require.Error(t, err)
	require.Equal(t, "OAUTH_CONFIG_INVALID", infraerrors.Reason(err))
}

func TestPublicSettingsExposeUniFedOAuthEnabled(t *testing.T) {
	svc := NewSettingService(&settingUniFedRepoStub{values: map[string]string{
		SettingKeyUniFedConnectEnabled: "true",
	}}, &config.Config{
		UniFed: config.UniFedConnectConfig{
			Enabled:     false,
			InstanceURL: "https://dc.hhhl.cc",
		},
	})

	settings, err := svc.GetPublicSettings(context.Background())

	require.NoError(t, err)
	require.True(t, settings.UniFedOAuthEnabled)
}
