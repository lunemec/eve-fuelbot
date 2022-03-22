package handler

import (
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

func (h *handler) loginHandler(w http.ResponseWriter, r *http.Request) error {
	state, err := uuid.NewV4()
	if err != nil {
		return errors.Wrap(err, "unable to create random state")
	}

	session, _ := h.store.New(r, "eve-quartermaster-session")
	session.Values["state"] = state.String()
	err = session.Save(r, w)
	if err != nil {
		return errors.Wrap(err, "unable to save session state")
	}

	// Generate the SSO URL with the state string
	url := h.sso.AuthorizeURL(state.String(), true, h.scopes)

	// Send the user to the URL
	http.Redirect(w, r, url, http.StatusFound)
	return nil
}
