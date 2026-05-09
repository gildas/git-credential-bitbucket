package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	core "github.com/gildas/go-core"
	logger "github.com/gildas/go-logger"
)

// DefaultRenewBefore tells when a token should be renewed with Bitbucket
const DefaultRenewBefore = 10 * time.Minute

func main() {
	// Analyzing the command line arguments
	var (
		storeLocation  = flag.String("store-location", core.GetEnvAsString("STORE_LOCATION", ""), "the location folder where the credentials data is stored")
		logDestination = flag.String("log", core.GetEnvAsString("LOG_DESTINATION", ""), "sends logs to the given destination. Default: none")
		logLevel       = flag.String("log-level", core.GetEnvAsString("LOG_LEVEL", "INFO"), "sets the log level. Default: info")
		workspace      = flag.String("workspace", core.GetEnvAsString("WORKSPACE", ""), "use the credentials for the given workspace. Default: none")
		renewBefore    = flag.Duration("renew", core.GetEnvAsDuration("RENEW_BEFORE", DefaultRenewBefore), "when to renew the bitbucket token. Default 10 minutes before it expires")
		version        = flag.Bool("version", false, "prints the current version and exits")
	)
	flag.Parse()

	if *version {
		fmt.Printf("%s version %s\n", APP, Version())
		os.Exit(0)
	}

	// Initializing the Logger
	var log *logger.Logger

	if len(*logDestination) > 0 {
		log = logger.Create(APP, *logDestination, logger.ParseLevel(*logLevel))
	} else if core.GetEnvAsBool("DEBUG", false) {
		log = logger.Create(APP, &logger.FileStream{
			Path:         filepath.Join(".", "log", APP+".log"),
			FilterLevels: logger.ParseLevels(*logLevel),
			Unbuffered:   true,
		})
	} else {
		log = logger.Create(APP, &logger.NilStream{})
	}
	defer log.Flush()
	log.Infof("%s", strings.Repeat("-", 80))
	log.Infof("Starting %s v. %s", APP, VERSION)
	log.Infof("Log Destination: %s", log)
	mainctx := log.ToContext(context.Background())

	// Creating the store folder as needed
	if len(*storeLocation) == 0 {
		*storeLocation = filepath.Join(core.GetEnvAsString("XDG_DATA_HOME", filepath.Join(core.GetEnvAsString("HOME", "."), ".local", "share")), APP)
	}
	*storeLocation = path.Clean(*storeLocation)
	if _, err := os.Stat(*storeLocation); os.IsNotExist(err) {
		if err = os.MkdirAll(*storeLocation, os.ModePerm); err != nil {
			log.Fatalf("Failed to create the storage folder", err)
			fmt.Fprintf(os.Stderr, "Failed to create the storage location: %s. Error: %s\n", *storeLocation, err.Error())
			log.Close()
			os.Exit(-1)
		}
	}
	log.Infof("Store Location: %s", *storeLocation)
	log.Infof("Token should be renewed %s before it expires", *renewBefore)

	// Command parameters come from stdin
	parameters := map[string]string{}
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			break
		}
		log.Debugf("Command line: %s", line)
		components := strings.Split(line, "=")
		if len(components) > 1 {
			key := strings.TrimSpace(components[0])
			value := strings.TrimSpace(strings.Join(components[1:], "="))
			log.Debugf("Adding Parameter[%s] = %s", key, value)
			parameters[key] = value
		} else {
			log.Warnf("Ignoring mal-formed entry: %s", line)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Failed to read stdin", err)
		fmt.Fprintf(os.Stderr, "Cannot read stdin for parameters. Error: %s\n", err.Error())
		log.Close()
		os.Exit(-1)
	}
	if len(*workspace) > 0 {
		log.Infof("Adding Parameter[workspace] = %s", *workspace)
		parameters["workspace"] = *workspace
	} else {
		log.Infof("No workspace specified")
	}

	log.Infof("Command: %s", flag.Arg(0))
	switch strings.ToLower(flag.Arg(0)) {
	case "get":
		credentials, err := LoadCredentials(mainctx, path.Clean(*storeLocation), parameters)
		if err != nil {
			log.Errorf("Failed to load credentials", err)
			fmt.Fprintf(os.Stderr, "Failed to load credentials. Error: %s\n", err)
			os.Exit(-1)
		}
		log.Record("credentials", credentials).Debugf("Found credentials")
		currentToken := credentials.Token
		if err = credentials.GetToken(mainctx, *renewBefore); err != nil {
			log.Errorf("Failed to get token for credentials", err)
			fmt.Fprintf(os.Stderr, "Failed to get token for credentials. Error: %s\n", err)
			os.Exit(-1)
		}
		if currentToken == nil || currentToken.AccessToken != credentials.Token.AccessToken {
			if err = credentials.Save(mainctx, path.Clean(*storeLocation)); err != nil {
				log.Errorf("Failed to save credentials", err)
			}
		}
		credentials.Fprint(os.Stdout)
	case "store":
		if username, found := parameters["username"]; found && username == "x-token-auth" {
			log.Debugf("git just tried to set the password for magic user %s, ignoring", username)
			os.Exit(0)
		}
		if _, found := parameters["password"]; found {
			log.Debugf("git just tried to set the password with the token, ignoring")
			os.Exit(0)
		}
		if _, err := CreateCredentials(mainctx, path.Clean(*storeLocation), parameters); err != nil {
			log.Errorf("Failed to create credentials", err)
			fmt.Fprintf(os.Stderr, "Failed to create credentials. Error: %s\n", err)
			os.Exit(-1)
		}
	case "erase":
		if _, found := parameters["password"]; found {
			log.Debugf("git just tried to clear the password, ignoring")
			os.Exit(0)
		}
		if err := DeleteCredentials(mainctx, path.Clean(*storeLocation), parameters); err != nil {
			log.Errorf("Failed to delete credentials", err)
			fmt.Fprintf(os.Stderr, "Failed to delete credentials. Error: %s\n", err)
			os.Exit(-1)
		}
	default:
		log.Warnf("Unsupported command: %s", flag.Arg(0))
	}
	os.Exit(0)
}
