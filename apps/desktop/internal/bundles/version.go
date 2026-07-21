package bundles

import (
	"strconv"
	"strings"
)

// CurrentAppVersion is compared against a bundle's MinDropzoneVersion
// header at load time; main sets it from its build version at startup.
var CurrentAppVersion = "0.0.0"

// VersionNewer reports whether version a is strictly newer than b, using
// dotted-numeric comparison (a "v" prefix and any prerelease suffix are
// ignored).
func VersionNewer(a, b string) bool {
	pa, pb := versionParts(a), versionParts(b)
	for i := 0; i < 3; i++ {
		if pa[i] != pb[i] {
			return pa[i] > pb[i]
		}
	}
	return false
}

func versionParts(v string) [3]int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	var parts [3]int
	for i, s := range strings.SplitN(v, ".", 3) {
		digits := s
		for j, r := range s {
			if r < '0' || r > '9' {
				digits = s[:j]
				break
			}
		}
		parts[i], _ = strconv.Atoi(digits)
	}
	return parts
}
