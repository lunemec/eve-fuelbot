package handler

import (
	"encoding/gob"
	"net/http"
	"os"

	"github.com/lunemec/eve-fuelbot/pkg/token"

	"github.com/antihax/goesi"
	"github.com/antihax/goesi/esi"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

type handler struct {
	signalChan   chan os.Signal
	log          handlerLogger
	tokenStorage token.Storage
	esi          *goesi.APIClient
	sso          *goesi.SSOAuthenticator
	router       http.Handler
	store        *sessions.CookieStore
	scopes       []string

	cache cache
}

type nameCache map[int32]esi.GetUniverseTypesTypeIdOk
type cache struct {
	names nameCache
}

type handlerLogger interface {
	Infow(string, ...interface{})
	Errorw(string, ...interface{})
}

func init() {
	gob.Register(goesi.VerifyResponse{})
	gob.Register(oauth2.Token{})
}

// New constructs new API http handler.
func New(signalChan chan os.Signal, log handlerLogger, client *http.Client, tokenStorage token.Storage, secretKey []byte, clientID, ssoSecret string, callbackURL string, scopes []string) http.Handler {
	esi := goesi.NewAPIClient(client, "EVE Quartermaster (lu.nemec@gmail.com)")
	sso := goesi.NewSSOAuthenticatorV2(client, clientID, ssoSecret, callbackURL, scopes)
	r := chi.NewRouter()
	h := handler{
		signalChan:   signalChan,
		log:          log,
		tokenStorage: tokenStorage,
		esi:          esi,
		sso:          sso,
		router:       r,
		store:        sessions.NewCookieStore(secretKey),
		scopes:       scopes,
		cache: cache{
			names: make(nameCache),
		},
	}

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/", ErrorHandler(h.indexHandler, h.log))
	r.Get("/login", ErrorHandler(h.loginHandler, h.log))
	r.Get("/callback", ErrorHandler(h.callbackHandler, h.log))
	return &h
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func (h *handler) session(r *http.Request) *sessions.Session {
	sess, _ := h.store.Get(r, "eve-quartermaster-session")
	return sess
}

func (h *handler) tokenSource(r *http.Request, w http.ResponseWriter) (oauth2.TokenSource, error) {
	session := h.session(r)
	token, ok := session.Values["token"].(oauth2.Token)
	if !ok {
		return nil, errors.Errorf("no token saved in session")
	}

	ts := h.sso.TokenSource(&token)
	newToken, err := ts.Token()
	if err != nil {
		return nil, errors.Wrapf(err, "error getting token")
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

func (h *handler) character(r *http.Request) (*goesi.VerifyResponse, error) {
	session := h.session(r)
	char, ok := session.Values["character"].(goesi.VerifyResponse)
	if !ok {
		return nil, errors.New("unable to get character from session")
	}
	return &char, nil
}
