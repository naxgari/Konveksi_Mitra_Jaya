package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/anggyputra/go-class/internal/model"
	"github.com/anggyputra/go-class/internal/service"
)

// PekerjaForm menyimpan data untuk halaman formulir.
type PekerjaForm struct {
	Judul          string
	Action         string
	Pekerja        model.Pekerja
	ModeEdit       bool
	UpahAwal       string
	KeteranganUpah string
	Error          string
}

// PekerjaHandler menangani permintaan HTTP modul pekerja.
type PekerjaHandler struct {
	Svc  *service.PekerjaService
	View *Renderer
}

// Daftar menampilkan seluruh data pekerja.
func (h *PekerjaHandler) Daftar(w http.ResponseWriter, r *http.Request) {
	daftar, err := h.Svc.Daftar(r.Context())
	if err != nil {
		gagal(w, err)
		return
	}

	h.View.Tampil(w, "pekerja/list.html", daftar)
}

// Detail menampilkan informasi seorang pekerja.
func (h *PekerjaHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}

	p, err := h.Svc.Detail(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}

	h.View.Tampil(w, "pekerja/detail.html", p)
}

// FormTambah menampilkan formulir tambah pekerja.
func (h *PekerjaHandler) FormTambah(w http.ResponseWriter, r *http.Request) {
	h.View.Tampil(w, "pekerja/form.html", PekerjaForm{
		Judul:  "Form Tambah Data Pekerja",
		Action: "/pekerja",
	})
}

// Simpan memproses data pekerja baru beserta upah awalnya.
func (h *PekerjaHandler) Simpan(w http.ResponseWriter, r *http.Request) {
	upahAwal := strings.TrimSpace(r.PostFormValue("upah_awal"))
	ketUpahStr := strings.TrimSpace(r.PostFormValue("keterangan_upah"))
	var ketUpah *string
	if ketUpahStr != "" {
		ketUpah = &ketUpahStr
	}

	p, err := pekerjaDariForm(r)
	if err != nil {
		var ve *service.ValidasiError
		if errors.As(err, &ve) {
			h.View.TampilStatus(w, http.StatusBadRequest, "pekerja/form.html", PekerjaForm{
				Judul:          "Form Tambah Data Pekerja",
				Action:         "/pekerja",
				Pekerja:        p,
				UpahAwal:       upahAwal,
				KeteranganUpah: ketUpahStr,
				Error:          ve.Pesan,
			})
			return
		}
		gagal(w, err)
		return
	}

	if _, err := h.Svc.BuatDenganUpahAwal(r.Context(), p, upahAwal, ketUpah); err != nil {
		var ve *service.ValidasiError
		if errors.As(err, &ve) {
			h.View.TampilStatus(w, http.StatusBadRequest, "pekerja/form.html", PekerjaForm{
				Judul:          "Form Tambah Data Pekerja",
				Action:         "/pekerja",
				Pekerja:        p,
				UpahAwal:       upahAwal,
				KeteranganUpah: ketUpahStr,
				Error:          ve.Pesan,
			})
			return
		}
		gagal(w, err)
		return
	}

	http.Redirect(w, r, "/pekerja", http.StatusSeeOther)
}

// FormEdit menampilkan formulir edit pekerja.
func (h *PekerjaHandler) FormEdit(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}

	p, err := h.Svc.Detail(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}

	h.View.Tampil(w, "pekerja/form.html", PekerjaForm{
		Judul:    "Form Edit Data Pekerja",
		Action:   fmt.Sprintf("/pekerja/%d/edit", id),
		Pekerja:  p,
		ModeEdit: true,
	})
}

// Update memproses perubahan data pekerja.
func (h *PekerjaHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}

	action := fmt.Sprintf("/pekerja/%d/edit", id)
	p, err := pekerjaDariForm(r)
	p.ID = id
	if err != nil {
		var ve *service.ValidasiError
		if errors.As(err, &ve) {
			h.View.TampilStatus(w, http.StatusBadRequest, "pekerja/form.html", PekerjaForm{
				Judul:    "Form Edit Data Pekerja",
				Action:   action,
				Pekerja:  p,
				ModeEdit: true,
				Error:    ve.Pesan,
			})
			return
		}
		gagal(w, err)
		return
	}

	if err := h.Svc.Ubah(r.Context(), p); err != nil {
		var ve *service.ValidasiError
		if errors.As(err, &ve) {
			h.View.TampilStatus(w, http.StatusBadRequest, "pekerja/form.html", PekerjaForm{
				Judul:    "Form Edit Data Pekerja",
				Action:   action,
				Pekerja:  p,
				ModeEdit: true,
				Error:    ve.Pesan,
			})
			return
		}
		gagal(w, err)
		return
	}

	http.Redirect(w, r, "/pekerja", http.StatusSeeOther)
}

// Nonaktifkan mencatat tanggal keluar pekerja.
func (h *PekerjaHandler) Nonaktifkan(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}

	tanggal := strings.TrimSpace(r.PostFormValue("tanggal_keluar"))

	tanggalKeluar, err := time.Parse("2006-01-02", tanggal)
	if err != nil {
		gagal(w, &service.ValidasiError{
			Pesan: "Tanggal keluar tidak valid",
		})
		return
	}

	if err := h.Svc.Nonaktifkan(r.Context(), id, tanggalKeluar); err != nil {
		gagal(w, err)
		return
	}

	http.Redirect(w, r, "/pekerja", http.StatusSeeOther)
}

// pekerjaDariForm membaca data dari formulir HTML.
func pekerjaDariForm(r *http.Request) (model.Pekerja, error) {
	tanggal := strings.TrimSpace(r.PostFormValue("tanggal_masuk"))

	var tanggalMasuk time.Time

	if tanggal != "" {
		var err error

		tanggalMasuk, err = time.Parse("2006-01-02", tanggal)
		if err != nil {
			return model.Pekerja{}, &service.ValidasiError{
				Pesan: "Format tanggal masuk tidak valid",
			}
		}
	}

	kontak := strings.TrimSpace(r.PostFormValue("kontak"))
	var kontakPekerja *string

	if kontak != "" {
		kontakPekerja = &kontak
	}

	return model.Pekerja{
		Nama:         r.PostFormValue("nama"),
		JenisPekerja: r.PostFormValue("jenis_pekerja"),
		Jabatan:      r.PostFormValue("jabatan"),
		Kontak:       kontakPekerja,
		TanggalMasuk: tanggalMasuk,
	}, nil
}
