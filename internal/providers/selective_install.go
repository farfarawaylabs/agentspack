package providers

import (
	"fmt"
	"os"
	"strings"

	"github.com/agentspack/agentspack/internal/wizard"
)

func selectedSet(items []string) map[string]struct{} {
	if len(items) == 0 {
		return nil
	}

	set := make(map[string]struct{}, len(items))
	for _, item := range items {
		normalized := strings.TrimSpace(item)
		if normalized == "" {
			continue
		}
		set[normalized] = struct{}{}
	}
	return set
}

func isSelected(set map[string]struct{}, name string) bool {
	if set == nil {
		return true
	}
	_, ok := set[name]
	return ok
}

func writeFileWithConflictPolicy(outputPath string, content []byte, conflictMessage string, policy wizard.ConflictPolicy) (bool, error) {
	if _, err := os.Stat(outputPath); err == nil {
		if policy == wizard.ConflictPolicySkipExisting {
			fmt.Printf("  Skipped existing: %s\n", outputPath)
			return false, nil
		}
		return false, fmt.Errorf("%s", conflictMessage)
	} else if !os.IsNotExist(err) {
		return false, err
	}

	if err := os.WriteFile(outputPath, content, 0644); err != nil {
		return false, err
	}

	fmt.Printf("  Created: %s\n", outputPath)
	return true, nil
}
