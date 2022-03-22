package handler

import (
	"net/http"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

func (h *handler) indexHandler(w http.ResponseWriter, r *http.Request) error {
	_, err := h.character(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return nil
	}
	_, err = h.tokenSource(r, w)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return nil
	}

	session := h.session(r)
	token := session.Values["token"].(oauth2.Token)
	err = h.tokenStorage.Write(token)
	if err != nil {
		return errors.Wrap(err, "unable to save token")
	}
	_, _ = w.Write([]byte("logged in successfully"))
	// Spawn a goroutine that will send SIGTEM in 1s.
	go func() {
		time.Sleep(1 * time.Second)
		h.signalChan <- syscall.SIGTERM
	}()
	return nil
}
