package goversion

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	Recommended  = "1.26"
	Conservative = "1.25"
	MinimumCLI   = "1.22"
)

var versionPattern = regexp.MustCompile(`^1\.([0-9]+)$`)

type Support string

const (
	Latest      Support = "latest"
	Supported   Support = "supported"
	Unsupported Support = "unsupported"
	Future      Support = "future"
)

func Validate(version string) error {
	if !versionPattern.MatchString(version) {
		return fmt.Errorf("Go version %q must use major.minor format, for example %s", version, Recommended)
	}
	return nil
}

func Classify(version string) Support {
	match := versionPattern.FindStringSubmatch(version)
	if len(match) != 2 {
		return Unsupported
	}
	minor, _ := strconv.Atoi(match[1])
	switch {
	case minor == 26:
		return Latest
	case minor == 25:
		return Supported
	case minor > 26:
		return Future

	default:
		return Unsupported
	}
}

func Description(version string) string {
	switch Classify(version) {
	case Latest:
		return "recommended for new projects; latest supported release line"
	case Supported:
		return "supported conservative baseline"
	case Future:
		return "newer than this GOKUB release policy; verify your toolchain and dependencies"
	default:
		return "unsupported by Go upstream; plan an upgrade"
	}
}

func ParseGoMod(content string) string {
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "go" && Validate(fields[1]) == nil {
			return fields[1]
		}
	}
	return ""
}
