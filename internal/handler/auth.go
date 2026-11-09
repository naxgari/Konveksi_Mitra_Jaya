package handler

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/anggyputra/go-class/internal/middleware"
	"github.com/anggyputra/go-class/internal/model"
	"github.com/anggyputra/go-class/internal/service"
)

type LoginPageData struct {
	Error      string
	Sukses     string
	Username   string
}

type RegisterPageData struct {
	Error       string
	Username    string
	NamaLengkap string
	Peran       string
}

type AuthHandler struct {
	Svc          *service.AuthService
	View         *Renderer
	CookieSecure bool
}

func (h *AuthHandler) FormLogin(w http.ResponseWriter, r *http.Request) {
	if h.sudahLogin(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	sukses := ""
	if r.URL.Query().Get("registered") == "1" {
		sukses = "Registrasi akun berhasil! Silakan masuk menggunakan akun baru Anda."
	} else if r.URL.Query().Get("logout") == "1" {
		sukses = "Anda telah berhasil keluar dari sistem."
	}

	h.View.Tampil(w, "auth/login.html", LoginPageData{
		Sukses: sukses,
	})
}

func (h *AuthHandler) ProsesLogin(w http.ResponseWriter, r *http.Request) {
	username := r.PostFormValue("username")
	password := r.PostFormValue("password")

	sesi, err := h.Svc.Login(r.Context(), username, password)
	if err != nil {
		pesan := "Terjadi kesalahan pada server"
		var ve *service.ValidasiError
		switch {
		case errors.As(err, &ve):
			pesan = ve.Pesan
		case errors.Is(err, model.ErrKredensialSalah):
			pesan = "Username atau password yang Anda masukkan salah."
		default:
			log.Printf("error login: %v", err)
		}

		w.WriteHeader(http.StatusUnauthorized)
		h.View.Tampil(w, "auth/login.html", LoginPageData{
			Error:    pesan,
			Username: username,
		})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.NamaCookieSesi,
		Value:    sesi.Token,
		Path:     "/",
		Expires:  sesi.ExpiredAt,
		MaxAge:   int(time.Until(sesi.ExpiredAt).Seconds()),
		HttpOnly: true,
		Secure:   h.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandler) FormRegister(w http.ResponseWriter, r *http.Request) {
	h.View.Tampil(w, "auth/register.html", RegisterPageData{
		Peran: string(model.PeranAdmin),
	})
}

func (h *AuthHandler) ProsesRegister(w http.ResponseWriter, r *http.Request) {
	in := service.RegisterInput{
		Username:           r.PostFormValue("username"),
		NamaLengkap:        r.PostFormValue("nama_lengkap"),
		Password:           r.PostFormValue("password"),
		KonfirmasiPassword: r.PostFormValue("konfirmasi_password"),
		Peran:              r.PostFormValue("peran"),
	}

	_, err := h.Svc.Register(r.Context(), in)
	if err != nil {
		pesan := "Gagal mendaftarkan pengguna"
		var ve *service.ValidasiError
		if errors.As(err, &ve) {
			pesan = ve.Pesan
		} else {
			log.Printf("error register: %v", err)
		}

		w.WriteHeader(http.StatusBadRequest)
		h.View.Tampil(w, "auth/register.html", RegisterPageData{
			Error:       pesan,
			Username:    in.Username,
			NamaLengkap: in.NamaLengkap,
			Peran:       in.Peran,
		})
		return
	}

	http.Redirect(w, r, "/login?registered=1", http.StatusSeeOther)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(middleware.NamaCookieSesi); err == nil && c.Value != "" {
		_ = h.Svc.Logout(r.Context(), c.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.NamaCookieSesi,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/login?logout=1", http.StatusSeeOther)
}

func (h *AuthHandler) sudahLogin(r *http.Request) bool {
	c, err := r.Cookie(middleware.NamaCookieSesi)
	if err != nil || c.Value == "" {
		return false
	}
	_, err = h.Svc.VerifikasiSesi(r.Context(), c.Value)
	return err == nil
}

