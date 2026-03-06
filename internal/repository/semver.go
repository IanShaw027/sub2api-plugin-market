package repository

import "golang.org/x/mod/semver"

// ensureVPrefix adds a "v" prefix if missing, as required by golang.org/x/mod/semver.
func ensureVPrefix(s string) string {
	if s == "" {
		return ""
	}
	if s[0] != 'v' {
		return "v" + s
	}
	return s
}

// semverLTE returns true if a <= b using semantic versioning.
// Returns true if either string is not valid semver (fail-open for backward compatibility).
func semverLTE(a, b string) bool {
	va, vb := ensureVPrefix(a), ensureVPrefix(b)
	if !semver.IsValid(va) || !semver.IsValid(vb) {
		return a <= b
	}
	return semver.Compare(va, vb) <= 0
}

// semverGTE returns true if a >= b using semantic versioning.
// Returns true if either string is not valid semver (fail-open for backward compatibility).
func semverGTE(a, b string) bool {
	va, vb := ensureVPrefix(a), ensureVPrefix(b)
	if !semver.IsValid(va) || !semver.IsValid(vb) {
		return a >= b
	}
	return semver.Compare(va, vb) >= 0
}

// isVersionCompatible checks if a plugin version is compatible with the given host API version.
// Rule: min_api_version <= hostVersion AND (max_api_version == "" OR max_api_version >= hostVersion)
func isVersionCompatible(minAPIVersion, maxAPIVersion, hostVersion string) bool {
	if !semverLTE(minAPIVersion, hostVersion) {
		return false
	}
	if maxAPIVersion != "" && !semverGTE(maxAPIVersion, hostVersion) {
		return false
	}
	return true
}
