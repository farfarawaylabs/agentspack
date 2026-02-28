package providers

import (
	"fmt"

	"github.com/agentspack/agentspack/internal/wizard"
)

type SkillInvocationSettings struct {
	DisableModelInvocation *bool
	UserInvocable          *bool
}

func resolveInvocationSettings(profile wizard.SkillInvocationProfile, providerName string) (SkillInvocationSettings, string) {
	switch profile {
	case wizard.SkillInvocationManualOnly:
		if providerName == "codex" {
			return SkillInvocationSettings{}, "Warning: invocation profile 'manual-only' for codex requires agents/openai.yaml policy, which is not generated yet; using platform defaults"
		}
		return SkillInvocationSettings{
			DisableModelInvocation: boolPtr(true),
		}, ""
	case wizard.SkillInvocationAutoOnly:
		if providerName == "claude-code" {
			return SkillInvocationSettings{
				UserInvocable: boolPtr(false),
			}, ""
		}
		return SkillInvocationSettings{}, fmt.Sprintf(
			"Warning: invocation profile 'auto-only' is not fully supported for %s yet; using platform defaults",
			providerName,
		)
	case wizard.SkillInvocationDual, "":
		return SkillInvocationSettings{}, ""
	default:
		return SkillInvocationSettings{}, fmt.Sprintf(
			"Warning: unknown invocation profile '%s'; using defaults",
			profile,
		)
	}
}

func boolPtr(v bool) *bool {
	return &v
}
