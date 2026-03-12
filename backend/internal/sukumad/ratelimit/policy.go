package ratelimit

func ScopeLabel(policy Policy) string {
	if policy.ScopeRef == "" {
		return policy.ScopeType
	}
	return policy.ScopeType + ":" + policy.ScopeRef
}
