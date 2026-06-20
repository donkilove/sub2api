package service

const (
	UserLimitSourceUser  = "user"
	UserLimitSourceGroup = "group"
	UserLimitSourceNone  = "none"
)

type EffectiveIntLimit struct {
	Limit  int
	Source string
}

// ResolveEffectiveUserConcurrencyLimit resolves the runtime user concurrency cap.
// User override wins over group defaults. Nil override means inherit; 0 means unlimited.
func ResolveEffectiveUserConcurrencyLimit(user *User, group *Group) EffectiveIntLimit {
	if user != nil && user.UserConcurrencyOverride != nil {
		return EffectiveIntLimit{Limit: *user.UserConcurrencyOverride, Source: UserLimitSourceUser}
	}
	if group != nil && group.UserConcurrencyLimit > 0 {
		return EffectiveIntLimit{Limit: group.UserConcurrencyLimit, Source: UserLimitSourceGroup}
	}
	return EffectiveIntLimit{Limit: 0, Source: UserLimitSourceNone}
}

// ResolveEffectiveUserRPMLimit resolves the runtime RPM cap.
// User override wins over group defaults. Nil override means inherit; 0 means unlimited.
func ResolveEffectiveUserRPMLimit(user *User, group *Group) EffectiveIntLimit {
	if user != nil && user.UserRPMLimitOverride != nil {
		return EffectiveIntLimit{Limit: *user.UserRPMLimitOverride, Source: UserLimitSourceUser}
	}
	if group != nil && group.RPMLimit > 0 {
		return EffectiveIntLimit{Limit: group.RPMLimit, Source: UserLimitSourceGroup}
	}
	return EffectiveIntLimit{Limit: 0, Source: UserLimitSourceNone}
}
