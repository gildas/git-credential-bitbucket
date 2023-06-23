package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	core "github.com/gildas/go-core"
	errors "github.com/gildas/go-errors"
	logger "github.com/gildas/go-logger"
	request "github.com/gildas/go-request"
)

// Credentials describes Bitbucket credentials
type Credentials struct {
	Protocol  string         `json:"protocol"`
	Host      string         `json:"host"`
	Username  string         `json:"username"`
	Workspace string         `json:"workspace,omitempty"`
	ClientID  string         `json:"client_id"`
	Secret    string         `json:"secret"`
	Token     *Token         `json:"token,omitempty"`
	Logger    *logger.Logger `json:"-"`
}

// Token describes a Bitbucket Token
type Token struct {
	TokenType    string    `json:"token_type"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	Expires      time.Time `json:"-"`
	Scopes       string    `json:"scopes"`
}

func (token Token) MarshalJSON() (payload []byte, err error) {
	type surrogate Token
	payload, err = json.Marshal(struct {
		surrogate
		ExpiresOn core.Time `json:"expires_on"`
	}{
		surrogate: surrogate(token),
		ExpiresOn: (core.Time)(token.Expires),
	})
	return payload, errors.JSONMarshalError.Wrap(err)
}

func (token *Token) UnmarshalJSON(payload []byte) (err error) {
	type surrogate Token
	var inner struct {
		surrogate
		ExpiresOn *core.Time `json:"expires_on"`
		ExpiresIn int64      `json:"expires_in"`
	}

	if err = json.Unmarshal(payload, &inner); err != nil {
		return errors.JSONUnmarshalError.Wrap(err)
	}
	*token = Token(inner.surrogate)
	if inner.ExpiresOn != nil {
		token.Expires = inner.ExpiresOn.AsTime()
	} else {
		token.Expires = time.Now().Add(time.Duration(inner.ExpiresIn) * time.Second)
	}
	return
}

// NewCredentials instantiates new Credentials from a map
func NewCredentials(parameters map[string]string, log *logger.Logger) (*Credentials, error) {
	var merr errors.MultiError
	credentials := &Credentials{
		Logger: logger.CreateIfNil(log, APP).Child("credentials", "credentials"),
	}
	if value, ok := parameters["protocol"]; ok {
		credentials.Protocol = value
	} else {
		credentials.Protocol = "https"
	}
	if value, ok := parameters["host"]; ok {
		credentials.Host = value
	} else {
		merr.Append(errors.ArgumentMissing.With("host"))
	}
	if value, ok := parameters["username"]; ok {
		credentials.Username = value
	} else {
		merr.Append(errors.ArgumentMissing.With("username"))
	}
	if value, ok := parameters["workspace"]; ok {
		credentials.Workspace = value
	}
	return credentials, merr.AsError()
}

// NewCredentials instantiates new Credentials from a map
//
// client id and secrets are expected
func NewCredentialsWithSecrets(parameters map[string]string, log *logger.Logger) (*Credentials, error) {
	var merr errors.MultiError

	credentials, err := NewCredentials(parameters, log)
	if err != nil {
		merr.Append(err)
	}
	if value, ok := parameters["clientid"]; ok {
		credentials.ClientID = value
	} else {
		merr.Append(errors.ArgumentMissing.With("clientid"))
	}
	if value, ok := parameters["secret"]; ok {
		credentials.Secret = value
	} else {
		merr.Append(errors.ArgumentMissing.With("secret"))
	}
	if value, ok := parameters["workspace"]; ok {
		credentials.Workspace = value
	}
	return credentials, merr.AsError()
}

// CreateCredentials creates new credentials in the store
func CreateCredentials(path string, parameters map[string]string, log *logger.Logger) (*Credentials, error) {
	credentials, err := NewCredentialsWithSecrets(parameters, log)
	if err != nil {
		return nil, err
	}
	if err := credentials.Save(path); err != nil {
		return nil, err
	}
	return credentials, nil
}

// LoadCredentials loads Credentials from the store
func LoadCredentials(path string, parameters map[string]string, log *logger.Logger) (*Credentials, error) {
	credentials, err := NewCredentials(parameters, log)
	if err != nil {
		return nil, err
	}
	filename := filepath.Join(path, credentials.Filename())
	credentials.Logger.Child(nil, "load").Debugf("Loading from %s", filename)
	payload, err := os.ReadFile(filename)
	if err != nil {
		return nil, errors.NotFound.With("file", credentials.Username)
	}
	err = json.Unmarshal(payload, &credentials)
	return credentials, errors.JSONUnmarshalError.Wrap(err)
}

// Save saves Credentials to the store
func (credentials Credentials) Save(path string) error {
	payload, err := json.Marshal(credentials)
	if err != nil {
		return errors.JSONMarshalError.Wrap(err)
	}
	filename := filepath.Join(path, credentials.Filename())
	credentials.Logger.Child(nil, "save").Debugf("Saving into %s", filename)
	return os.WriteFile(filename, payload, 0600)
}

// DeleteCredentials delete Credentials from the store
func DeleteCredentials(path string, parameters map[string]string) error {
	credentials, err := NewCredentials(parameters, nil)
	if err != nil {
		return nil
	}
	filename := filepath.Join(path, credentials.Filename())
	credentials.Logger.Child(nil, "delete").Debugf("Deleting %s", filename)
	return os.Remove(filename)
}

// Filename gives the filename used to load/save the Credentials from/to the store
func (credentials Credentials) Filename() string {
	if len(credentials.Workspace) > 0 {
		return fmt.Sprintf("%s-%s@%s.json", credentials.Username, credentials.Workspace, credentials.Host)
	}
	return fmt.Sprintf("%s@%s.json", credentials.Username, credentials.Host)
}

func (credentials *Credentials) GetToken(renewBefore time.Duration) error {
	log := credentials.Logger.Child(nil, "gettoken")
	now := time.Now()
	if credentials.Token == nil || now.After(credentials.Token.Expires) {
		if credentials.Token != nil {
			log.Infof("Token expired %s ago (On: %s)", now.Sub(credentials.Token.Expires), credentials.Token.Expires)
		}
		token := &Token{}
		authURL, _ := url.Parse("https://bitbucket.org/site/oauth2/access_token")
		if _, err := request.Send(
			&request.Options{
				Method:        http.MethodPost,
				URL:           authURL,
				Authorization: request.BasicAuthorization(credentials.ClientID, credentials.Secret),
				Payload: map[string]string{
					"grant_type": "client_credentials",
				},
				UserAgent: fmt.Sprintf("%s v%s", APP, VERSION),
				Logger:    credentials.Logger,
			},
			&token,
		); err != nil {
			return err
		}
		credentials.Token = token
	} else if credentials.Token != nil && now.After(credentials.Token.Expires.Add(-1*renewBefore)) {
		renewOn := credentials.Token.Expires.Add(-1 * renewBefore)
		log.Infof("Token is still valid, but expires in %s (On: %s), we should renew the token", now.Sub(renewOn), renewOn)
		token := &Token{}
		authURL, _ := url.Parse("https://bitbucket.org/site/oauth2/access_token")
		if _, err := request.Send(
			&request.Options{
				Method:        http.MethodPost,
				URL:           authURL,
				Authorization: request.BasicAuthorization(credentials.ClientID, credentials.Secret),
				Payload: map[string]string{
					"grant_type":    "refresh_token",
					"refresh_token": credentials.Token.RefreshToken,
				},
				UserAgent: fmt.Sprintf("%s v%s", APP, VERSION),
				Logger:    credentials.Logger,
			},
			&token,
		); err != nil {
			return err
		}
		credentials.Token = token
	} else {
		log.Infof("Token is still valid and expires in %s (On: %s)", credentials.Token.Expires.Sub(now), credentials.Token.Expires)
	}
	return nil
}

// Fprint prints Credentials for git to consume
func (credentials Credentials) Fprint(out *os.File) {
	fmt.Fprintf(out, "protocol=%s\n", credentials.Protocol)
	fmt.Fprintf(out, "host=%s\n", credentials.Host)
	fmt.Fprintf(out, "username=%s\n", "x-token-auth")
	if credentials.Token != nil {
		fmt.Fprintf(out, "password=%s\n", credentials.Token.AccessToken)
	}
}
