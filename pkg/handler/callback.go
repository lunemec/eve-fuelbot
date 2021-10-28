package handler

import (
	"net/http"

	"github.com/pkg/errors"
)

func (h *handler) callbackHandler(w http.ResponseWriter, r *http.Request) error {
	// get our code and state
	code := r.FormValue("code")
	state := r.FormValue("state")
	session := h.session(r)

	if session.Values["state"] != nil && session.Values["state"] != state {
		h.log.Errorw("state mismatch", "stored", session.Values["state"], "received", state)
		return errors.New("state mismatch, login again")
	}

	// Exchange the code for an Access and Refresh token.
	token, err := h.sso.TokenExchange(code)
	if err != nil {
		return errors.Wrap(err, "token exchange error")
	}

	// Obtain a token source (automaticlly pulls refresh as needed)
	tokSrc := h.sso.TokenSource(token)

	// Verify the client (returns clientID)
	v, err := h.sso.Verify(tokSrc)
	if err != nil {
		return errors.Wrap(err, "token verify error")
	}

	token, err = tokSrc.Token()
	if err != nil {
		return errors.Wrap(err, "token source error getting new token")
	}
	// Save token.
	session.Values["token"] = *token

	// Save the verification structure on the session for quick access.
	session.Values["character"] = v
	err = session.Save(r, w)
	if err != nil {
		return errors.Wrap(err, "unable to save session")
	}

	h.log.Infow("token verified", "v", v)
	http.Redirect(w, r, "/", 302)
	return nil
}
