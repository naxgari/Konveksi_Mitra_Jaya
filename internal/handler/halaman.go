package handler

import (
	"net/http"

	"github.com/anggyputra/go-class/internal/middleware"
	"github.com/anggyputra/go-class/internal/model"
	"github.com/anggyputra/go-class/internal/service"
)

type HomeData struct {
	Pengguna  model.Pengguna
	Ringkasan model.RingkasanDashboard
}

type HalamanHandler struct {
	View    *Renderer
	AuthSvc *service.AuthService
}

func (h *HalamanHandler) Home(w http.ResponseWriter, r *http.Request) {
	pengguna, _ := middleware.PenggunaDariContext(r.Context())
	var ringkasan model.RingkasanDashboard
	if h.AuthSvc != nil {
		ringkasan = h.AuthSvc.RingkasanDashboard(r.Context())
	}

	h.View.Tampil(w, "home.html", HomeData{
		Pengguna:  pengguna,
		Ringkasan: ringkasan,
	})
}

func (h *HalamanHandler) Profil(w http.ResponseWriter, r *http.Request) {
	pengguna, _ := middleware.PenggunaDariContext(r.Context())
	h.View.Tampil(w, "profil.html", pengguna)
}
