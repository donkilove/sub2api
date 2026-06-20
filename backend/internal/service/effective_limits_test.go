//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveEffectiveUserConcurrencyLimit(t *testing.T) {
	groupLimit := &Group{ID: 10, UserConcurrencyLimit: 4}

	require.Equal(t, EffectiveIntLimit{Limit: 4, Source: UserLimitSourceGroup},
		ResolveEffectiveUserConcurrencyLimit(&User{ID: 1}, groupLimit))

	userLimit := 2
	require.Equal(t, EffectiveIntLimit{Limit: 2, Source: UserLimitSourceUser},
		ResolveEffectiveUserConcurrencyLimit(&User{ID: 1, UserConcurrencyOverride: &userLimit}, groupLimit))

	unlimited := 0
	require.Equal(t, EffectiveIntLimit{Limit: 0, Source: UserLimitSourceUser},
		ResolveEffectiveUserConcurrencyLimit(&User{ID: 1, UserConcurrencyOverride: &unlimited}, groupLimit))

	require.Equal(t, EffectiveIntLimit{Limit: 0, Source: UserLimitSourceNone},
		ResolveEffectiveUserConcurrencyLimit(&User{ID: 1}, &Group{ID: 10}))
}

func TestResolveEffectiveUserRPMLimit(t *testing.T) {
	groupLimit := &Group{ID: 10, RPMLimit: 60}

	require.Equal(t, EffectiveIntLimit{Limit: 60, Source: UserLimitSourceGroup},
		ResolveEffectiveUserRPMLimit(&User{ID: 1}, groupLimit))

	userLimit := 20
	require.Equal(t, EffectiveIntLimit{Limit: 20, Source: UserLimitSourceUser},
		ResolveEffectiveUserRPMLimit(&User{ID: 1, UserRPMLimitOverride: &userLimit}, groupLimit))

	unlimited := 0
	require.Equal(t, EffectiveIntLimit{Limit: 0, Source: UserLimitSourceUser},
		ResolveEffectiveUserRPMLimit(&User{ID: 1, UserRPMLimitOverride: &unlimited}, groupLimit))

	require.Equal(t, EffectiveIntLimit{Limit: 0, Source: UserLimitSourceNone},
		ResolveEffectiveUserRPMLimit(&User{ID: 1}, &Group{ID: 10}))
}
