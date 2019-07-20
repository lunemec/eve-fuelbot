package version

import (
	"fmt"
)

// These variables are filled using `govvv` tool using -ldflags. Do not modify them.
var (
	// BuildDate ... well guess.
	BuildDate = ""
	// GitCommit of this binary.
	GitCommit = ""
	// Version is, suprisingly version.
	Version = ""
)

// These variables can be modified.
var (
	versionString = `EVE FuelBot v%s
Built: %s @ %s`
	// VersionString is used when `gateway version` is called.
	VersionString = fmt.Sprintf(versionString, Version, BuildDate, GitCommit)
)
