package middleware

import (
	"context"
	"net/http"
	"slices"

	"github.com/anggyputra/go-class/internal/model"
	"github.com/anggyputra/go-class/internal/service"
)

const NamaCookieSesi = "session_token"

type ctxKeyPengguna struct{}

// UserResponseWriter membungkus http.ResponseWriter agar Renderer otomatis mengetahui pengguna yang sedang login.
type UserResponseWriter struct {
	http.ResponseWriter
	Pengguna *model.Pengguna
}

func (w *UserResponseWriter) CurrentPengguna() *model.Pengguna {
	return w.Pengguna
}

func SimpanPenggunaKeContext(ctx context.Context, u model.Pengguna) context.Context {
	return context.WithValue(ctx, ctxKeyPengguna{}, u)
}

func PenggunaDariContext(ctx context.Context) (model.Pengguna, bool) {
	u, ok := ctx.Value(ctxKeyPengguna{}).(model.Pengguna)
	return u, ok
}

// RequireLogin memastikan pengguna sudah memiliki cookie sesi yang valid sebelum mengakses route privat.
func RequireLogin(authSvc *service.AuthService, cookieSecure bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(NamaCookieSesi)
		if err != nil || c.Value == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		pengguna, err := authSvc.VerifikasiSesi(r.Context(), c.Value)
		if err != nil {
			// Hapus cookie yang sudah tidak valid/kedaluwarsa
			http.SetCookie(w, &http.Cookie{
				Name:     NamaCookieSesi,
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				HttpOnly: true,
				Secure:   cookieSecure,
				SameSite: http.SameSiteLaxMode,
			})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		ctx := SimpanPenggunaKeContext(r.Context(), pengguna)
		urw := &UserResponseWriter{
			ResponseWriter: w,
			Pengguna:       &pengguna,
		}
		next.ServeHTTP(urw, r.WithContext(ctx))
	})
}

// RequireRole membatasi akses handler hanya untuk daftar peran (role) tertentu.
func RequireRole(roles ...model.PeranPengguna) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			pengguna, ok := PenggunaDariContext(r.Context())
			if !ok {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			if !slices.Contains(roles, pengguna.Peran) {
				http.Error(w, "403 Forbidden — Akses ditolak: Peran Anda ("+pengguna.LabelPeran()+") tidak memiliki izin untuk mengakses fitur ini.", http.StatusForbidden)
				return
			}

			next(w, r)
		}
	}
}

