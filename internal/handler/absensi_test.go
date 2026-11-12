package handler

import (
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/anggyputra/go-class/internal/middleware"
	"github.com/anggyputra/go-class/internal/model"
)

func TestAbsensiTemplatesRenderForAdminAndPemilik(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("lokasi file test tidak diketahui")
	}
	t.Chdir(filepath.Join(filepath.Dir(file), "..", ".."))
	rd := NewRenderer()

	masuk := time.Date(2026, 11, 12, 7, 5, 0, 0, time.Local)
	keluar := time.Date(2026, 11, 12, 15, 30, 0, 0, time.Local)
	jadwal := model.JadwalShift{
		ID: 21, IDPekerja: 4, IDShift: 2, NamaPekerja: "Siti",
		JenisPekerja: "harian", NamaShift: "Pagi", JamMulai: "07:00", JamSelesai: "15:00",
		Tanggal: time.Date(2026, 11, 12, 0, 0, 0, 0, time.Local),
		Absensi: &model.Absensi{ID: 8, IDJadwal: 21, Status: "hadir", JamMasuk: &masuk, JamKeluar: &keluar, MenitTerlambat: 5, MenitLembur: 30, Keterangan: "Mesin terlambat"},
	}
	shift := model.Shift{ID: 2, Nama: "Pagi", JamMulai: "07:00", JamSelesai: "15:00"}
	pekerja := model.PekerjaJadwal{ID: 4, Nama: "Siti", Jenis: "harian"}
	templates := []struct {
		file string
		data any
		want string
	}{
		{"absensi/jadwal_list.html", JadwalPage{Admin: true, Jadwal: []model.JadwalShift{jadwal}, Pekerja: []model.PekerjaJadwal{pekerja}}, "Siti"},
		{"absensi/jadwal_form.html", JadwalFormPage{Judul: "Edit Jadwal", Action: "/jadwal/21/edit", FormPekerja: "4", FormShift: "2", Tanggal: "2026-11-12", Pekerja: []model.PekerjaJadwal{pekerja}, Shift: []model.Shift{shift}}, "2026-11-12"},
		{"absensi/absensi_form.html", AbsensiFormPage{Judul: "Catat Absensi", Action: "/jadwal/21/absensi", Jadwal: jadwal, Status: "perlu_diperiksa", JamMasuk: "07:05", JamKeluar: "15:30", Keterangan: "Mesin terlambat", Error: "Jam belum lengkap"}, "Jam belum lengkap"},
		{"absensi/pemeriksaan.html", PemeriksaanPage{Admin: true, Jadwal: []model.JadwalShift{jadwal}, Pekerja: []model.PekerjaJadwal{pekerja}, Shift: []model.Shift{shift}}, "5 menit"},
		{"absensi/shift_list.html", ShiftPage{Admin: true, Shift: []model.Shift{shift}}, "Pagi"},
		{"absensi/shift_form.html", ShiftFormPage{Judul: "Edit Shift", Action: "/shift/2/edit", Nama: "Pagi", JamMulai: "07:00", JamSelesai: "15:00", Error: "Nama shift sudah digunakan"}, "Nama shift sudah digunakan"},
	}
	admin := model.Pengguna{ID: 1, Peran: model.PeranAdmin}
	for _, item := range templates {
		t.Run(item.file, func(t *testing.T) {
			for _, user := range []model.Pengguna{admin, {ID: 2, Peran: model.PeranPemilik}} {
				rec := httptest.NewRecorder()
				rd.Tampil(&middleware.UserResponseWriter{ResponseWriter: rec, Pengguna: &user}, item.file, item.data)
				if rec.Code != 200 {
					t.Fatalf("%s status=%d", item.file, rec.Code)
				}
				if !strings.Contains(rec.Body.String(), item.want) {
					t.Errorf("%s tidak menampilkan %q", item.file, item.want)
				}
			}
		})
	}

	ownerResponse := httptest.NewRecorder()
	owner := model.Pengguna{ID: 2, Peran: model.PeranPemilik}
	rd.Tampil(&middleware.UserResponseWriter{ResponseWriter: ownerResponse, Pengguna: &owner}, "absensi/jadwal_list.html", JadwalPage{Admin: false, Jadwal: []model.JadwalShift{jadwal}})
	if strings.Contains(ownerResponse.Body.String(), "Tempatkan Shift") || strings.Contains(ownerResponse.Body.String(), "/jadwal/21/edit") {
		t.Fatal("Pemilik tidak boleh melihat kontrol mutasi jadwal")
	}
	if !strings.Contains(ownerResponse.Body.String(), "Baca saja") {
		t.Fatal("Pemilik harus melihat keterangan baca saja")
	}
}

func TestStatusAbsensiMembedakanBelumDicatatDanPerluDiperiksa(t *testing.T) {
	if got := statusAbsensi(model.JadwalShift{}); got != "belum_dicatat" {
		t.Fatalf("status tanpa catatan=%q", got)
	}
	a := &model.Absensi{Status: "perlu_diperiksa"}
	if got := statusAbsensi(model.JadwalShift{Absensi: a}); got != "perlu_diperiksa" {
		t.Fatalf("status belum lengkap=%q", got)
	}
	if err := validasiFilterTanggal(url.Values{"mulai": {"2026-11-13"}, "akhir": {"2026-11-12"}}); err == nil {
		t.Fatal("rentang tanggal terbalik seharusnya ditolak")
	}
}
