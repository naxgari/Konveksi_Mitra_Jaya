package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/anggyputra/go-class/internal/middleware"
	"github.com/anggyputra/go-class/internal/model"
	"github.com/anggyputra/go-class/internal/service"
)

type PenggajianHandler struct {
	PeriodeSvc *service.PeriodeGajiService
	GajiSvc    *service.GajiService
	View       *Renderer
}

type PeriodeListPage struct {
	Admin   bool
	Periode []model.PeriodeGaji
}

type PeriodeDetailPage struct {
	Admin, Pemilik bool
	Periode        model.PeriodeGaji
	Gaji           []model.Gaji
	Pekerja        []model.PekerjaGaji
	Error          string
	Pesan          string
}

type GajiDetailPage struct {
	Admin   bool
	Periode model.PeriodeGaji
	Gaji    model.Gaji
	Rincian []model.DetailGaji
	Error   string
}

type PeriodeFormPage struct {
	Jenis               string
	TanggalAwal         string
	TanggalAkhir        string
	TanggalBayarRencana string
	Error               string
}

func (h *PenggajianHandler) PeriodeDaftar(
	w http.ResponseWriter,
	r *http.Request,
) {
	pengguna, _ := middleware.PenggunaDariContext(r.Context())
	if !pengguna.AdalahAdmin() && !pengguna.AdalahPemilik() {
		http.Error(w, "Akses ditolak", http.StatusForbidden)
		return
	}

	daftar, err := h.PeriodeSvc.Daftar(r.Context())
	if err != nil {
		gagal(w, err)
		return
	}

	h.View.Tampil(w, "gaji/periode_list.html", PeriodeListPage{
		Admin:   pengguna.AdalahAdmin(),
		Periode: daftar,
	})
}

func (h *PenggajianHandler) PeriodeDetail(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	h.renderPeriodeDetail(w, r, id, http.StatusOK, "")
}

func (h *PenggajianHandler) renderPeriodeDetail(
	w http.ResponseWriter,
	r *http.Request,
	id int,
	status int,
	pesanError string,
) {
	pengguna, _ := middleware.PenggunaDariContext(r.Context())
	if !pengguna.AdalahAdmin() && !pengguna.AdalahPemilik() {
		http.Error(w, "Akses ditolak", http.StatusForbidden)
		return
	}

	periode, err := h.PeriodeSvc.Detail(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}

	daftar, err := h.GajiSvc.DaftarPerPeriode(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}

	var pekerja []model.PekerjaGaji

	if pengguna.AdalahAdmin() && periode.Status == "terbuka" {
		pekerja, err = h.GajiSvc.PekerjaPerPeriode(r.Context(), id)
		if err != nil {
			gagal(w, err)
			return
		}
	}

	h.View.TampilStatus(w, status, "gaji/periode_detail.html", PeriodeDetailPage{
		Admin:   pengguna.AdalahAdmin(),
		Pemilik: pengguna.AdalahPemilik(),
		Periode: periode,
		Gaji:    daftar,
		Pekerja: pekerja,
		Error:   pesanError,
		Pesan:   r.URL.Query().Get("pesan"),
	})
}

func (h *PenggajianHandler) GajiDetail(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	h.renderGajiDetail(w, r, id, http.StatusOK, "")
}

func (h *PenggajianHandler) renderGajiDetail(
	w http.ResponseWriter,
	r *http.Request,
	id int,
	status int,
	pesanError string,
) {
	pengguna, _ := middleware.PenggunaDariContext(r.Context())
	if !pengguna.AdalahAdmin() && !pengguna.AdalahPemilik() {
		http.Error(w, "Akses ditolak", http.StatusForbidden)
		return
	}

	gaji, err := h.GajiSvc.Detail(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}

	periode, err := h.PeriodeSvc.Detail(r.Context(), gaji.IDPeriode)
	if err != nil {
		gagal(w, err)
		return
	}

	rincian, err := h.GajiSvc.Rincian(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}

	h.View.TampilStatus(w, status, "gaji/gaji_detail.html", GajiDetailPage{
		Admin:   pengguna.AdalahAdmin(),
		Periode: periode,
		Gaji:    gaji,
		Rincian: rincian,
		Error:   pesanError,
	})
}

func (h *PenggajianHandler) PeriodeFormTambah(w http.ResponseWriter, r *http.Request) {
	pengguna, _ := middleware.PenggunaDariContext(r.Context())
	if !pengguna.AdalahAdmin() {
		http.Error(w, "Akses ditolak", http.StatusForbidden)
		return
	}

	h.View.Tampil(w, "gaji/periode_form.html", PeriodeFormPage{
		Jenis: "mingguan",
	})
}

func (h *PenggajianHandler) PeriodeSimpan(w http.ResponseWriter, r *http.Request) {
	pengguna, _ := middleware.PenggunaDariContext(r.Context())
	if !pengguna.AdalahAdmin() {
		http.Error(w, "Akses ditolak", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form tidak valid", http.StatusBadRequest)
		return
	}

	form := PeriodeFormPage{
		Jenis:               r.PostForm.Get("jenis"),
		TanggalAwal:         r.PostForm.Get("tanggal_awal"),
		TanggalAkhir:        r.PostForm.Get("tanggal_akhir"),
		TanggalBayarRencana: r.PostForm.Get("tanggal_bayar_rencana"),
	}

	awal, errAwal := time.ParseInLocation(
		"2006-01-02", form.TanggalAwal, time.Local,
	)
	akhir, errAkhir := time.ParseInLocation(
		"2006-01-02", form.TanggalAkhir, time.Local,
	)
	bayar, errBayar := time.ParseInLocation(
		"2006-01-02", form.TanggalBayarRencana, time.Local,
	)

	if errAwal != nil || errAkhir != nil || errBayar != nil {
		form.Error = "Semua tanggal wajib diisi dengan tanggal yang valid"
		h.View.TampilStatus(w, http.StatusBadRequest, "gaji/periode_form.html", form)
		return
	}

	_, err := h.PeriodeSvc.Buat(r.Context(), model.PeriodeGaji{
		Jenis:               form.Jenis,
		TanggalAwal:         awal,
		TanggalAkhir:        akhir,
		TanggalBayarRencana: bayar,
	})
	if err != nil {
		if !validasi(err) {
			gagal(w, err)
			return
		}

		form.Error = validasiPesan(err)
		h.View.TampilStatus(w, http.StatusBadRequest, "gaji/periode_form.html", form)
		return
	}

	http.Redirect(w, r, "/periode", http.StatusSeeOther)
}

func (h *PenggajianHandler) HitungDraf(
	w http.ResponseWriter,
	r *http.Request,
) {
	pengguna, _ := middleware.PenggunaDariContext(r.Context())
	if !pengguna.AdalahAdmin() {
		http.Error(w, "Akses ditolak", http.StatusForbidden)
		return
	}

	idPeriode, ok := idDariPath(w, r)
	if !ok {
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderPeriodeDetail(w, r, idPeriode, http.StatusBadRequest, "Form tidak valid")
		return
	}

	idPekerja, err := strconv.Atoi(r.PostForm.Get("id_pekerja"))
	if err != nil || idPekerja <= 0 {
		h.renderPeriodeDetail(w, r, idPeriode, http.StatusBadRequest, "Pilih pekerja yang valid")
		return
	}

	idGaji, err := h.GajiSvc.HitungDraf(
		r.Context(),
		idPekerja,
		idPeriode,
		0,
		0,
	)
	if err != nil {
		if validasi(err) {
			h.renderPeriodeDetail(w, r, idPeriode, http.StatusBadRequest, validasiPesan(err))
			return
		}
		gagal(w, err)
		return
	}

	http.Redirect(
		w,
		r,
		"/gaji/"+strconv.FormatInt(idGaji, 10),
		http.StatusSeeOther,
	)
}

func (h *PenggajianHandler) HitungSemuaDraf(
	w http.ResponseWriter,
	r *http.Request,
) {
	pengguna, _ := middleware.PenggunaDariContext(r.Context())
	if !pengguna.AdalahAdmin() {
		http.Error(w, "Akses ditolak", http.StatusForbidden)
		return
	}

	idPeriode, ok := idDariPath(w, r)
	if !ok {
		return
	}

	pekerja, err := h.GajiSvc.PekerjaPerPeriode(r.Context(), idPeriode)
	if err != nil {
		gagal(w, err)
		return
	}
	if len(pekerja) == 0 {
		h.renderPeriodeDetail(w, r, idPeriode, http.StatusBadRequest, "Tidak ada pekerja yang perlu dihitung pada periode ini")
		return
	}

	berhasil := 0
	var kendala []string
	for _, p := range pekerja {
		if _, err := h.GajiSvc.HitungDraf(r.Context(), p.ID, idPeriode, 0, 0); err != nil {
			if validasi(err) {
				kendala = append(kendala, fmt.Sprintf("%s: %s", p.Nama, validasiPesan(err)))
				continue
			}
			gagal(w, err)
			return
		}
		berhasil++
	}

	if len(kendala) > 0 {
		msg := fmt.Sprintf("Berhasil menghitung %d pekerja. Kendala pada %d pekerja: %s", berhasil, len(kendala), strings.Join(kendala, "; "))
		h.renderPeriodeDetail(w, r, idPeriode, http.StatusBadRequest, msg)
		return
	}

	pesan := fmt.Sprintf("Berhasil menghitung draf gaji untuk semua %d pekerja pada periode ini.", berhasil)
	http.Redirect(w, r, "/periode/"+strconv.Itoa(idPeriode)+"?pesan="+url.QueryEscape(pesan), http.StatusSeeOther)
}


func (h *PenggajianHandler) BayarGaji(
	w http.ResponseWriter,
	r *http.Request,
) {
	pengguna, _ := middleware.PenggunaDariContext(r.Context())
	if !pengguna.AdalahAdmin() {
		http.Error(w, "Akses ditolak", http.StatusForbidden)
		return
	}

	id, ok := idDariPath(w, r)
	if !ok {
		return
	}

	gaji, err := h.GajiSvc.Detail(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}

	if err := h.GajiSvc.Bayar(
		r.Context(),
		pengguna.Peran,
		gaji.IDPekerja,
		gaji.IDPeriode,
	); err != nil {
		if validasi(err) {
			h.renderGajiDetail(w, r, id, http.StatusBadRequest, validasiPesan(err))
			return
		}
		gagal(w, err)
		return
	}

	http.Redirect(
		w,
		r,
		"/gaji/"+strconv.Itoa(id),
		http.StatusSeeOther,
	)
}

func (h *PenggajianHandler) TutupPeriode(
	w http.ResponseWriter,
	r *http.Request,
) {
	pengguna, _ := middleware.PenggunaDariContext(r.Context())
	if !pengguna.AdalahPemilik() {
		http.Error(w, "Akses ditolak", http.StatusForbidden)
		return
	}

	id, ok := idDariPath(w, r)
	if !ok {
		return
	}

	if err := h.PeriodeSvc.Tutup(
		r.Context(),
		pengguna,
		id,
	); err != nil {
		if validasi(err) {
			h.renderPeriodeDetail(w, r, id, http.StatusBadRequest, validasiPesan(err))
			return
		}
		gagal(w, err)
		return
	}

	http.Redirect(
		w,
		r,
		"/periode/"+strconv.Itoa(id),
		http.StatusSeeOther,
	)
}