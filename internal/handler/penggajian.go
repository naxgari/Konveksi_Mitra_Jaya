package handler

import (
	"net/http"
	"strconv"
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
}

type GajiDetailPage struct {
	Admin   bool
	Periode model.PeriodeGaji
	Gaji    model.Gaji
	Rincian []model.DetailGaji
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
	pengguna, _ := middleware.PenggunaDariContext(r.Context())
	if !pengguna.AdalahAdmin() && !pengguna.AdalahPemilik() {
		http.Error(w, "Akses ditolak", http.StatusForbidden)
		return
	}

	id, ok := idDariPath(w, r)
	if !ok {
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

	h.View.Tampil(w, "gaji/periode_detail.html", PeriodeDetailPage{
		Admin:   pengguna.AdalahAdmin(),
		Pemilik: pengguna.AdalahPemilik(),
		Periode: periode,
		Gaji:    daftar,
		Pekerja: pekerja,
	})
}

func (h *PenggajianHandler) GajiDetail(
	w http.ResponseWriter,
	r *http.Request,
) {
	pengguna, _ := middleware.PenggunaDariContext(r.Context())
	if !pengguna.AdalahAdmin() && !pengguna.AdalahPemilik() {
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

	h.View.Tampil(w, "gaji/gaji_detail.html", GajiDetailPage{
		Admin:   pengguna.AdalahAdmin(),
		Periode: periode,
		Gaji:    gaji,
		Rincian: rincian,
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
		w.WriteHeader(http.StatusBadRequest)
		h.View.Tampil(w, "gaji/periode_form.html", form)
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
		w.WriteHeader(http.StatusBadRequest)
		h.View.Tampil(w, "gaji/periode_form.html", form)
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
		http.Error(w, "Form tidak valid", http.StatusBadRequest)
		return
	}

	idPekerja, err := strconv.Atoi(r.PostForm.Get("id_pekerja"))
	if err != nil || idPekerja <= 0 {
		http.Error(w, "Pilih pekerja yang valid", http.StatusBadRequest)
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