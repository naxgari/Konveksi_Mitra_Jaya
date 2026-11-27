package handler

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

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
			TotalPekerjaAktif:        40,
			PekerjaHarianAktif:       30,
			PekerjaBulananAktif:      10,
			TotalGajiPeriodeBerjalan: "12500000.00",
			TotalSaldoKasbon:         "1500000.00",
		},
	})
	bodyAdmin := recHomeAdmin.Body.String()
	if !strings.Contains(bodyAdmin, "Data Pekerja &amp; Upah") {
		t.Errorf("sidebar Admin harus menampilkan menu Data Pekerja & Upah")
	}
	if !strings.Contains(bodyAdmin, "Rp12.500.000") || !strings.Contains(bodyAdmin, "Rp1.500.000") {
		t.Errorf("dashboard harus menampilkan angka KPI asli TotalGajiPeriodeBerjalan dan TotalSaldoKasbon")
	}

	// 3. Uji render halaman pekerja & slip untuk Pemilik (tombol Admin tidak boleh tampil)
	pemilikUser := model.Pengguna{
		ID:          2,
		Username:    "pemilik",
		NamaLengkap: "Bapak H. Hendra",
		Peran:       model.PeranPemilik,
		Aktif:       true,
	}
	recPekerjaPemilik := httptest.NewRecorder()
	urwPemilik := &middleware.UserResponseWriter{
		ResponseWriter: recPekerjaPemilik,
		Pengguna:       &pemilikUser,
	}
	rd.Tampil(urwPemilik, "pekerja/list.html", []model.Pekerja{{
		ID:           1,
		Nama:         "Ayu",
		JenisPekerja: "harian",
		Jabatan:      "Penjahit",
		TanggalMasuk: time.Now(),
	}})
	bodyPekerjaPemilik := recPekerjaPemilik.Body.String()
	if strings.Contains(bodyPekerjaPemilik, "+ Tambah Pekerja") || strings.Contains(bodyPekerjaPemilik, "Nonaktifkan") {
		t.Errorf("halaman pekerja untuk Pemilik TIDAK BOLEH menampilkan tombol Tambah atau Nonaktifkan")
	}

	recSlipPemilik := httptest.NewRecorder()
	rd.Tampil(&middleware.UserResponseWriter{ResponseWriter: recSlipPemilik, Pengguna: &pemilikUser}, "kasbon/slip.html", SlipPage{
		Slip:  model.SlipGaji{BisaPotong: true},
		Admin: false,
	})
	if strings.Contains(recSlipPemilik.Body.String(), "hitung-komponen") {
		t.Errorf("slip untuk Pemilik TIDAK BOLEH menampilkan tombol hitung-komponen")
	}

	pekerjaSample := model.Pekerja{ID: 1, Nama: "Ayu", JenisPekerja: "harian", Jabatan: "Penjahit", TanggalMasuk: time.Now(), CreatedAt: time.Now()}
	pages := []struct {
		template string
		data     any
	}{
		{"pekerja/list.html", []model.Pekerja{pekerjaSample}},
		{"pekerja/form.html", PekerjaForm{Judul: "Tambah", Action: "/pekerja"}},
		{"pekerja/detail.html", pekerjaSample},
		{"pekerja/upah_list.html", RiwayatUpahPage{Pekerja: pekerjaSample}},
		{"pekerja/upah_form.html", RiwayatUpahForm{Judul: "Tambah Upah", Action: "/pekerja/1/upah", Pekerja: pekerjaSample}},
		{"gaji/periode_list.html", PeriodeListPage{Admin: true}},
		{"gaji/periode_form.html", PeriodeFormPage{Jenis: "mingguan"}},
		{"gaji/periode_detail.html", PeriodeDetailPage{Admin: true, Periode: model.PeriodeGaji{ID: 1, Jenis: "mingguan", Status: "terbuka", TanggalAwal: time.Now(), TanggalAkhir: time.Now(), TanggalBayarRencana: time.Now()}}},
		{"gaji/gaji_detail.html", GajiDetailPage{Admin: true, Periode: model.PeriodeGaji{ID: 1, Status: "terbuka", TanggalAwal: time.Now(), TanggalAkhir: time.Now()}, Gaji: model.Gaji{ID: 1, IDPekerja: 1, NamaPekerja: "Ayu", GajiPokok: "720000.00", TotalLembur: "52500.00", PotonganTelat: "15000.00", TotalPotonganKasbon: "231750.00", TotalKoreksi: "0.00", GajiBruto: "772500.00", GajiBersih: "525750.00", StatusBayar: "belum"}}},
		{"kasbon/kasbon_list.html", KasbonDaftarPage{}},
		{"kasbon/kasbon_form.html", KasbonFormPage{}},
		{"kasbon/kasbon_detail.html", KasbonDetailPage{Kasbon: model.Kasbon{Tanggal: time.Now()}}},
		{"kasbon/koreksi_list.html", KoreksiDaftarPage{}},
		{"kasbon/koreksi_form.html", KoreksiFormPage{}},
		{"kasbon/slip.html", SlipPage{Slip: model.SlipGaji{BisaPotong: true}, Admin: true}},
		{"kasbon/rekap.html", RekapPage{}},
	}
	for _, page := range pages {
		t.Run(page.template, func(t *testing.T) {
			rec := httptest.NewRecorder()
			rd.Tampil(&middleware.UserResponseWriter{ResponseWriter: rec, Pengguna: &adminUser}, page.template, page.data)
			if rec.Code != http.StatusOK {
				t.Fatalf("render %s status = %d, want 200: %s", page.template, rec.Code, rec.Body.String())
			}
		})
	}
}
