package repository

import (
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestNormalizeEmailAuthIdentitySubjectRejectsUniFedSyntheticEmail(t *testing.T) {
	require.Empty(t, normalizeEmailAuthIdentitySubject("unifed-abc123"+service.UniFedConnectSyntheticEmailDomain))
}

func TestUserSignupSourceOrDefaultAllowsOAuthProviders(t *testing.T) {
	for _, provider := range []string{"linuxdo", "wechat", "oidc", "github", "google", "dingtalk", "unifed"} {
		require.Equal(t, provider, userSignupSourceOrDefault(" "+strings.ToUpper(provider)+" "))
	}
	require.Equal(t, "email", userSignupSourceOrDefault(""))
	require.Equal(t, "email", userSignupSourceOrDefault("unknown"))
}
