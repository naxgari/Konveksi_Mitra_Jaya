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

// RiwayatUpahPage berisi data halaman daftar riwayat upah.
type RiwayatUpahPage struct {
	Pekerja model.Pekerja
	Riwayat []model.RiwayatUpah
}

// RiwayatUpahForm berisi data halaman tambah upah.
type RiwayatUpahForm struct {
	Judul          string
	Action         string
	Pekerja        model.Pekerja
	TanggalBerlaku string
	Nominal        string
	Keterangan     string
	PesanError     string
}

// RiwayatUpahHandler menangani HTTP modul riwayat upah.
type RiwayatUpahHandler struct {
	Svc        *service.RiwayatUpahService
	PekerjaSvc *service.PekerjaService
	View       *Renderer
}

// Daftar menampilkan riwayat upah seorang pekerja.
func (h *RiwayatUpahHandler) Daftar(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}

	p, err := h.PekerjaSvc.Detail(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}

	riwayat, err := h.Svc.Daftar(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}

	h.View.Tampil(
		w,
		"pekerja/upah_list.html",
		RiwayatUpahPage{
			Pekerja: p,
			Riwayat: riwayat,
		},
	)
}

// FormTambah menampilkan formulir tarif upah baru.
func (h *RiwayatUpahHandler) FormTambah(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}

	p, err := h.PekerjaSvc.Detail(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}

	h.View.Tampil(
		w,
		"pekerja/upah_form.html",
		RiwayatUpahForm{
			Judul:   "Tambah Riwayat Upah",
			Action:  fmt.Sprintf("/pekerja/%d/upah", id),
			Pekerja: p,
		},
	)
}

// Simpan memproses penambahan tarif upah baru.
func (h *RiwayatUpahHandler) Simpan(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}

	// Mengambil data pekerja.
	p, err := h.PekerjaSvc.Detail(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}

	// Menyimpan input formulir agar tidak hilang
	// ketika validasi gagal.
	form := RiwayatUpahForm{
		Judul:          "Tambah Riwayat Upah",
		Action:         fmt.Sprintf("/pekerja/%d/upah", id),
		Pekerja:        p,
		TanggalBerlaku: strings.TrimSpace(r.PostFormValue("berlaku_mulai")),
		Nominal:        strings.TrimSpace(r.PostFormValue("nominal")),
		Keterangan:     strings.TrimSpace(r.PostFormValue("keterangan")),
	}

	// Membaca data formulir.
	u, err := riwayatUpahDariForm(r)

	if err == nil {
		u.IDPekerja = id

		// Menyimpan riwayat upah ke database.
		_, err = h.Svc.Buat(r.Context(), u)
	}

	// Menangani kesalahan penyimpanan.
	if err != nil {
		var validasiErr *service.ValidasiError

		if errors.As(err, &validasiErr) {
			// Menampilkan pesan validasi pada formulir.
			form.PesanError = validasiErr.Pesan

			w.WriteHeader(http.StatusBadRequest)

			h.View.Tampil(
				w,
				"pekerja/upah_form.html",
				form,
			)
			return
		}

		// Kesalahan lain ditangani seperti sebelumnya.
		gagal(w, err)
		return
	}

	// Jika berhasil, kembali ke daftar riwayat upah.
	http.Redirect(
		w,
		r,
		fmt.Sprintf("/pekerja/%d/upah", id),
		http.StatusSeeOther,
	)
}

// riwayatUpahDariForm membaca data formulir HTML.
func riwayatUpahDariForm(
	r *http.Request,
) (model.RiwayatUpah, error) {

	tanggal := strings.TrimSpace(
		r.PostFormValue("berlaku_mulai"),
	)

	tanggalBerlaku, err := time.Parse(
		"2006-01-02",
		tanggal,
	)

	if err != nil {
		return model.RiwayatUpah{},
			&service.ValidasiError{
				Pesan: "Tanggal berlaku tidak valid",
			}
	}

	nominal := strings.TrimSpace(
		r.PostFormValue("nominal"),
	)

	keterangan := strings.TrimSpace(
		r.PostFormValue("keterangan"),
	)

	var ket *string

	if keterangan != "" {
		ket = &keterangan
	}

	return model.RiwayatUpah{
		BerlakuMulai: tanggalBerlaku,
		Nominal:      nominal,
		Keterangan:   ket,
	}, nil
}
