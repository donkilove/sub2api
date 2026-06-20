package repository

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestNormalizeEmailAuthIdentitySubjectRejectsUniFedSyntheticEmail(t *testing.T) {
	require.Empty(t, normalizeEmailAuthIdentitySubject("unifed-abc123"+service.UniFedConnectSyntheticEmailDomain))
}
