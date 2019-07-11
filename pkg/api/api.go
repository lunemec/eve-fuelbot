package api

import (
	"context"
	"encoding/gob"
	"net/http"
	"time"

	"github.com/antihax/goesi"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/gofrs/uuid"
	"github.com/gorilla/sessions"
	"github.com/gregjones/httpcache"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

type api struct {
	log    apiLogger
	esi    *goesi.APIClient
	sso    *goesi.SSOAuthenticator
	router http.Handler
	store  *sessions.CookieStore
	scopes []string
}

type apiLogger interface {
	Infow(string, ...interface{})
	Errorw(string, ...interface{})
}

func init() {
	gob.Register(goesi.VerifyResponse{})
	gob.Register(oauth2.Token{})
}

// New constructs new API http handler.
func New(log apiLogger, secretKey []byte, clientID, ssoSecret string) http.Handler {
	transport := httpcache.NewTransport(httpcache.NewMemoryCache())
	transport.Transport = &http.Transport{Proxy: http.ProxyFromEnvironment}
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}
	esi := goesi.NewAPIClient(client, "EVE Scanner (lu.nemec@gmail.com)")
	scopes := []string{"publicData", "esi-location.read_location.v1"}
	sso := goesi.NewSSOAuthenticator(client, clientID, ssoSecret, "http://localhost:3000/callback", scopes)
	r := chi.NewRouter()
	a := api{
		log:    log,
		esi:    esi,
		sso:    sso,
		router: r,
		store:  sessions.NewCookieStore(secretKey),
		scopes: scopes,
	}

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/", a.index)
	r.Get("/login", a.login)
	r.Get("/callback", a.callback)
	r.Get("/location", a.location)
	return &a
}

func (a *api) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}

func (a *api) index(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("root."))
}

func (a *api) login(w http.ResponseWriter, r *http.Request) {
	state, err := uuid.NewV4()
	if err != nil {
		a.log.Errorw("unable to create uuid", "err", err)
		http.Error(w, "unable to create random state", http.StatusInternalServerError)
		return
	}

	session := a.session(r)
	session.Values["state"] = state.String()
	err = session.Save(r, w)
	if err != nil {
		a.log.Errorw("unable to save session", "err", err)
		http.Error(w, "unable to save session", http.StatusInternalServerError)
		return
	}

	// Generate the SSO URL with the state string
	url := a.sso.AuthorizeURL(state.String(), true, a.scopes)

	// Send the user to the URL
	http.Redirect(w, r, url, 302)
	return
}

func (a *api) callback(w http.ResponseWriter, r *http.Request) {
	// get our code and state
	code := r.FormValue("code")
	state := r.FormValue("state")
	session := a.session(r)

	if session.Values["state"] != state {
		a.log.Errorw("state mismatch", "stored", session.Values["state"], "received", state)
		http.Error(w, "state mismatch, login again", http.StatusInternalServerError)
		return
	}

	// Exchange the code for an Access and Refresh token.
	token, err := a.sso.TokenExchange(code)
	if err != nil {
		a.log.Errorw("token exchange error", "err", err)
		http.Error(w, "token exchange error", http.StatusInternalServerError)
		return
	}

	// Obtain a token source (automaticlly pulls refresh as needed)
	tokSrc := a.sso.TokenSource(token)

	// Verify the client (returns clientID)
	v, err := a.sso.Verify(tokSrc)
	if err != nil {
		a.log.Errorw("token verify error", "err", err)
		http.Error(w, "token verify error", http.StatusInternalServerError)
		return
	}

	token, err = tokSrc.Token()
	if err != nil {
		a.log.Errorw("token creation error", "err", err)
		http.Error(w, "token creation error", http.StatusInternalServerError)
		return
	}
	// Save token.
	session.Values["token"] = *token

	// Save the verification structure on the session for quick access.
	session.Values["character"] = v
	err = session.Save(r, w)
	if err != nil {
		a.log.Errorw("unable to save session", "err", err)
		http.Error(w, "unable to save session", http.StatusInternalServerError)
		return
	}

	a.log.Infow("token verified", "v", v)
	http.Redirect(w, r, "/location", 302)
}

func (a *api) location(w http.ResponseWriter, r *http.Request) {
	tokenSrc, err := a.tokenSource(r, w)
	if err != nil {
		a.log.Errorw("error getting token", "err", err)
		http.Error(w, "error getting token, login again", http.StatusInternalServerError)
		return
	}
	ctx := context.WithValue(context.Background(), goesi.ContextOAuth2, tokenSrc)
	location, _, err := a.esi.ESI.LocationApi.GetCharactersCharacterIdLocation(ctx, int32(93227004), nil)
	if err != nil {
		a.log.Errorw("location read error", "err", err)
		http.Error(w, "location read error", http.StatusInternalServerError)
		return
	}
	a.log.Infow("location", "l", location)
	system, _, err := a.esi.ESI.UniverseApi.GetUniverseSystemsSystemId(ctx, location.SolarSystemId, nil)
	if err != nil {
		a.log.Errorw("system read error", "err", err)
		http.Error(w, "system read error", http.StatusInternalServerError)
		return
	}
	a.log.Infow("system", "s", system)
}

func (a *api) session(r *http.Request) *sessions.Session {
	sess, _ := a.store.Get(r, "session")
	return sess
}

func (a *api) tokenSource(r *http.Request, w http.ResponseWriter) (oauth2.TokenSource, error) {
	session := a.session(r)
	token, ok := session.Values["token"].(oauth2.Token)
	if !ok {
		return nil, errors.Errorf("no token saved in session")
	}

	ts := a.sso.TokenSource(&token)
	newToken, err := ts.Token()
	if err != nil {
		return nil, errors.Errorf("error getting token")
	}

	if token != *newToken {
		// Save token.
		session.Values["token"] = *newToken
		err = session.Save(r, w)
		if err != nil {
			return nil, errors.Wrap(err, "unable to save session")
		}
	}

	return ts, nil
}
