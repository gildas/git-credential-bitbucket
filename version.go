package main

// commit contains the current git commit and is set in the build.sh script
var commit string

// VERSION is the version of this application
var VERSION = "1.0.3" + commit

const (
	// APP is the name of the application
	APP = "git-credential-bitbucket"
)
