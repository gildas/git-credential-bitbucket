package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	core "github.com/gildas/go-core"
	logger "github.com/gildas/go-logger"
)

// Log is the application Logger
var Log *logger.Logger

// DefaultRenewBefore tells when a token should be renewed with Bitbucket
const DefaultRenewBefore = 10 * time.Minute

func main() {
	// Analyzing the command line arguments
	var (
		storeLocation  = flag.String("store-location", core.GetEnvAsString("STORE_LOCATION", ""), "the location folder where the credentials data is stored")
		logDestination = flag.String("log", core.GetEnvAsString("LOG_DESTINATION", ""), "sends logs to the given destination. Default: none")
		workspace      = flag.String("workspace", core.GetEnvAsString("WORKSPACE", ""), "use the credentials for the given workspace. Default: none")
		renewBefore    = flag.Duration("renew", core.GetEnvAsDuration("RENEW_BEFORE", DefaultRenewBefore), "when to renew the bitbucket token. Default 10 minutes before it expires")
		version        = flag.Bool("version", false, "prints the current version and exits")
	)
	flag.Parse()

	if *version {
		fmt.Printf("%s version %s\n", APP, VERSION)
		os.Exit(0)
	}

	// Initializing the Logger
	if len(*logDestination) > 0 {
		Log = logger.Create(APP, *logDestination)
	} else if core.GetEnvAsBool("DEBUG", false) {
		Log = logger.Create(APP, &logger.FileStream{
			Path:       filepath.Join(".", "log", APP+".log"),
			Unbuffered: true,
		})
	} else {
		Log = logger.Create(APP, &logger.NilStream{})
	}
	defer Log.Close()
	Log.Infof(strings.Repeat("-", 80))
	Log.Infof("Starting %s v. %s", APP, VERSION)
	Log.Infof("Log Destination: %s", Log)

	// Creating the store folder as needed
	if len(*storeLocation) == 0 {
		*storeLocation = filepath.Join(core.GetEnvAsString("XDG_DATA_HOME", filepath.Join(core.GetEnvAsString("HOME", "."), ".local", "share")), APP)
	}
	if _, err := os.Stat(*storeLocation); os.IsNotExist(err) {
		if err = os.MkdirAll(*storeLocation, os.ModePerm); err != nil {
			Log.Fatalf("Failed to create the storage folder", err)
			fmt.Fprintf(os.Stderr, "Failed to create the storage location: %s. Error: %s\n", *storeLocation, err.Error())
			Log.Close()
			os.Exit(-1)
		}
	}
	Log.Infof("Store Location: %s", *storeLocation)
	Log.Infof("Token should be renewd %s before it expires", *renewBefore)

	// Command parameters come from stdin
	parameters := map[string]string{}
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			break
		}
		components := strings.Split(line, "=")
		if len(components) > 1 {
			key := strings.TrimSpace(components[0])
			value := strings.TrimSpace(strings.Join(components[1:], "="))
			Log.Debugf("Adding Parameter[%s] = %s", key, value)
			parameters[key] = value
		} else {
			Log.Warnf("Ignoring mal-formed entry: %s", line)
		}
	}
	if err := scanner.Err(); err != nil {
		Log.Fatalf("Failed to read stdin", err)
		fmt.Fprintf(os.Stderr, "Cannot read stdin for parameters. Error: %s\n", err.Error())
		Log.Close()
		os.Exit(-1)
	}
	if len(*workspace) > 0 {
		Log.Debugf("Adding Parameter[workspace] = %s", *workspace)
		parameters["workspace"] = *workspace
	}

	Log.Infof("Command: %s", flag.Arg(0))
	switch strings.ToLower(flag.Arg(0)) {
	case "get":
		credentials, err := LoadCredentials(*storeLocation, parameters, Log)
		if err != nil {
			Log.Errorf("Failed to load credentials", err)
			fmt.Fprintf(os.Stderr, "Failed to load credentials. Error: %s\n", err)
			os.Exit(-1)
		}
		Log.Record("credentials", credentials).Debugf("Found credentials")
		if err = credentials.GetToken(*renewBefore); err != nil {
			Log.Errorf("Failed to get token for credentials", err)
			fmt.Fprintf(os.Stderr, "Failed to get token for credentials. Error: %s\n", err)
			os.Exit(-1)
		}
		if err = credentials.Save(*storeLocation); err != nil {
			Log.Errorf("Failed to save credentials", err)
		}
		credentials.Fprint(os.Stdout)
	case "store":
		if _, found := parameters["password"]; found {
			Log.Debugf("git just tried to reset the password, ignoring")
			os.Exit(0)
		}
		if _, err := CreateCredentials(*storeLocation, parameters, Log); err != nil {
			Log.Errorf("Failed to create credentials", err)
			fmt.Fprintf(os.Stderr, "Failed to create credentials. Error: %s\n", err)
			os.Exit(-1)
		}
	case "erase":
		if err := DeleteCredentials(*storeLocation, parameters); err != nil {
			Log.Errorf("Failed to deleting credentials", err)
			fmt.Fprintf(os.Stderr, "Failed to delete credentials. Error: %s\n", err)
			os.Exit(-1)
		}
	default:
		Log.Warnf("Unsupported command: %s", flag.Arg(0))
	}
	os.Exit(0)
}
