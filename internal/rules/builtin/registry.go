package builtin

import "github.com/platform-manager/platform-manager/internal/rules"

// GetAllRules returns all built-in rules
func GetAllRules() []rules.Rule {
	return []rules.Rule{
		// Pod rules
		&CrashLoopBackOffRule{},
		&ImagePullBackOffRule{},
		&PodPendingRule{},

		// Crossplane rules
		&CrossplaneFailedIAMRule{},
		&PausedButSyncingRule{},
		&ProviderUnhealthyRule{},
		&StaleResourceRule{},

		// ArgoCD rules
		&ArgoSyncFailedRule{},
		&ArgoOutOfSyncRule{},

		// Resource usage rules
		&HighResourceUsageRule{},

		// IAM rules (integrates with drift detection)
		&IAMExtraPrivilegesRule{},
	}
}

// GetRuleByID returns a rule by its ID
func GetRuleByID(id string) rules.Rule {
	for _, rule := range GetAllRules() {
		if rule.ID() == id {
			return rule
		}
	}
	return nil
}

// GetRulesByCategory returns rules grouped by category
func GetRulesByCategory() map[string][]rules.Rule {
	return map[string][]rules.Rule{
		"pods": {
			&CrashLoopBackOffRule{},
			&ImagePullBackOffRule{},
			&PodPendingRule{},
		},
		"crossplane": {
			&CrossplaneFailedIAMRule{},
			&PausedButSyncingRule{},
			&ProviderUnhealthyRule{},
			&StaleResourceRule{},
		},
		"argocd": {
			&ArgoSyncFailedRule{},
			&ArgoOutOfSyncRule{},
		},
		"resources": {
			&HighResourceUsageRule{},
		},
		"iam": {
			&IAMExtraPrivilegesRule{},
		},
	}
}

// GetCriticalRules returns only critical severity rules
func GetCriticalRules() []rules.Rule {
	var critical []rules.Rule
	for _, rule := range GetAllRules() {
		if rule.Severity() == rules.SeverityCritical {
			critical = append(critical, rule)
		}
	}
	return critical
}
