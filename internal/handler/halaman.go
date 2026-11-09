package handler

import "net/http"

type HalamanHandler struct {
	View *Renderer
}

func (h *HalamanHandler) Home(w http.ResponseWriter, r *http.Request) {
	h.View.Tampil(w, "home.html", nil)
}

func (h *HalamanHandler) Profil(w http.ResponseWriter, r *http.Request) {
	h.View.Tampil(w, "profil.html", nil)
}
