package install

// hookForms lists every command an install may have written for binaryPath:
// the current quoted form, and the raw form installs wrote before
// buildHookCmd quoted paths with spaces (#113). Identical when the path
// has no space.
func hookForms(binaryPath string) []string {
	quoted, raw := buildHookCmd(binaryPath), binaryPath+" hook"
	if quoted == raw {
		return []string{quoted}
	}
	return []string{quoted, raw}
}

func hasAnyHookForm(settings map[string]interface{}, binaryPath string) bool {
	for _, cmd := range hookForms(binaryPath) {
		if isHookInstalled(settings, cmd) {
			return true
		}
	}
	return false
}

func removeAllHookForms(settings map[string]interface{}, binaryPath string) bool {
	removed := false
	for _, cmd := range hookForms(binaryPath) {
		// Evaluate removeHook first so every form is removed, not just the first hit.
		removed = removeHook(settings, cmd) || removed
	}
	return removed
}
