package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsReservedEmail_DingTalkDomain(t *testing.T) {
	require.True(t, isReservedEmail("dingtalk-123@dingtalk-connect.invalid"))
	require.True(t, isReservedEmail("DINGTALK-456@DINGTALK-CONNECT.INVALID")) // case-insensitive
	require.False(t, isReservedEmail("real@dingtalk.com"))
}

func TestUniFedReservedEmailAndLegacySignupSource(t *testing.T) {
	require.True(t, isReservedEmail("unifed-abc123@unifed-connect.invalid"))
	require.True(t, isReservedEmail("UNIFED-ABC123@UNIFED-CONNECT.INVALID"))
	require.False(t, isReservedEmail("real@unifed.example"))
	require.Equal(t, "unifed", inferLegacySignupSource("unifed-abc123@unifed-connect.invalid"))
}
