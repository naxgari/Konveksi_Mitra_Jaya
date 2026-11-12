package handler

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/anggyputra/go-class/internal/middleware"
	"github.com/anggyputra/go-class/internal/model"
)

func TestRendererTemplates(t *testing.T) {
	// Pindah direktori kerja ke root project agar folder templates/ terbaca saat go test dijalankan
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("lokasi file test tidak diketahui")
	}
	t.Chdir(filepath.Join(filepath.Dir(testFile), "..", ".."))

	rd := NewRenderer()

	// 1. Uji render halaman login tanpa login (guest-body)
	recLogin := httptest.NewRecorder()
	rd.Tampil(recLogin, "auth/login.html", LoginPageData{
		Sukses: "Berhasil daftar",
	})
	if recLogin.Code != http.StatusOK {
		t.Fatalf("status login = %d, ingin 200", recLogin.Code)
	}
	bodyLogin := recLogin.Body.String()
	if !strings.Contains(bodyLogin, "Masuk ke Sistem Penggajian") {
		t.Errorf("halaman login tidak memuat judul yang diharapkan")
	}
	if !strings.Contains(bodyLogin, "guest-body") {
		t.Errorf("halaman login tamu harus menggunakan class guest-body")
	}

	// 2. Uji render halaman home sebagai Admin
	adminUser := model.Pengguna{
		ID:          1,
		Username:    "admin",
		NamaLengkap: "Muhammad Naufal A.",
		Peran:       model.PeranAdmin,
		Aktif:       true,
	}
	recHomeAdmin := httptest.NewRecorder()
	urwAdmin := &middleware.UserResponseWriter{
		ResponseWriter: recHomeAdmin,
		Pengguna:       &adminUser,
	}
	rd.Tampil(urwAdmin, "home.html", HomeData{
		Pengguna: adminUser,
		Ringkasan: model.RingkasanDashboard{
			TotalPekerjaAktif:   40,
			PekerjaHarianAktif:  30,
			PekerjaBulananAktif: 10,
		},
	})
	bodyAdmin := recHomeAdmin.Body.String()
	if !strings.Contains(bodyAdmin, "Data Pekerja &amp; Upah") {
		t.Errorf("sidebar Admin harus menampilkan menu Data Pekerja & Upah")
	}

	// 3. Uji render halaman home sebagai Petugas Absensi (Supervisor) -> Menu gaji harus tersembunyi
	supUser := model.Pengguna{
		ID:          2,
		Username:    "supervisor",
		NamaLengkap: "Muhammad Falih A.",
		Peran:       model.PeranSupervisor,
		Aktif:       true,
	}
	recHomeSup := httptest.NewRecorder()
	urwSup := &middleware.UserResponseWriter{
		ResponseWriter: recHomeSup,
		Pengguna:       &supUser,
	}
	rd.Tampil(urwSup, "home.html", HomeData{
		Pengguna: supUser,
	})
	bodySup := recHomeSup.Body.String()
	if strings.Contains(bodySup, "Periode &amp; Hitung Gaji") {
		t.Errorf("sidebar Petugas Absensi (Supervisor) TIDAK BOLEH menampilkan menu Periode & Hitung Gaji")
	}
}
