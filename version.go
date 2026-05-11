package main

import "strings"

// commit contains the current git commit and is set in the build.sh script
var commit string

// branch contains the git branch this code was built on and should be set via -ldflags
var branch string

// stamp contains the build date and should be set via -ldflags
var stamp string

// VERSION is the version of this application
var VERSION = "1.1.1"

// APP is the name of the application
const APP = "git-credential-bitbucket"

// PACKAGE is the name of the package (used to create artifacts)
const PACKAGE = "git-credential-bitbucket"

// Version gets the current version of the application
func Version() string {
	if strings.HasPrefix(strings.ToLower(branch), "dev") || strings.HasPrefix(strings.ToLower(branch), "feature") {
		return VERSION + "+" + stamp + "." + commit
	}
	return VERSION
}
