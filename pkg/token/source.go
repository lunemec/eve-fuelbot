package token

import (
	"net/http"

	"github.com/antihax/goesi"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

// Source is interface for token source.
type Source interface {
	Token() (*oauth2.Token, error)
	TokenSource() (oauth2.TokenSource, error)
	Verify() (*goesi.VerifyResponse, error)
}

type logger interface {
	Infow(string, ...interface{})
	Errorw(string, ...interface{})
}

type source struct {
	sso     *goesi.SSOAuthenticator
	storage Storage
}

// NewSource returns new token source from storage.
func NewSource(log logger, client *http.Client, storage Storage, secretKey []byte, clientID, ssoSecret string, callbackURL string, scopes []string) Source {
	sso := goesi.NewSSOAuthenticatorV2(client, clientID, ssoSecret, callbackURL, scopes)
	return &source{
		storage: storage,
		sso:     sso,
	}
}

func (s *source) Token() (*oauth2.Token, error) {
	ts, err := s.TokenSource()
	if err != nil {
		return nil, errors.Wrap(err, "unable to read token")
	}
	newToken, err := ts.Token()
	if err != nil {
		return nil, errors.Wrapf(err, "error getting token")
	}

	// Save token.
	err = s.storage.Write(*newToken)
	if err != nil {
		return nil, errors.Wrap(err, "unable to save refreshed token")
	}

	return newToken, nil
}

func (s *source) TokenSource() (oauth2.TokenSource, error) {
	token, err := s.storage.Read()
	if err != nil {
		return nil, errors.Wrap(err, "unable to read token")
	}
	return s.sso.TokenSource(&token), nil
}

func (s *source) Verify() (*goesi.VerifyResponse, error) {
	ts, err := s.TokenSource()
	if err != nil {
		return nil, errors.Wrap(err, "unable to create token source")
	}
	return s.sso.Verify(ts)
}
