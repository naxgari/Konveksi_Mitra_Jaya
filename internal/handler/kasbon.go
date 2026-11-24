package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/anggyputra/go-class/internal/middleware"
	"github.com/anggyputra/go-class/internal/model"
	"github.com/anggyputra/go-class/internal/service"
)

type KasbonHandler struct {
	Svc     *service.KasbonService
	GajiSvc *service.GajiService
	View    *Renderer
}

type KasbonDaftarPage struct {
	Daftar []model.Kasbon
	Admin  bool
}

type KasbonFormPage struct {
	Pekerja    []model.PekerjaJadwal
	IDPekerja  string
	Tanggal    string
	Jumlah     string
	Keterangan string
	Error      string
}

type KasbonDetailPage struct {
	Kasbon   model.Kasbon
	Potongan []model.PotonganKasbon
}

type KoreksiDaftarPage struct {
	Daftar  []model.KoreksiGaji
	Admin   bool
	Pemilik bool
}

type KoreksiFormPage struct {
	Gaji         []model.PilihanGaji
	IDGajiAsal   string
	Tanggal      string
	NilaiSelisih string
	Alasan       string
	Error        string
}

type SlipPage struct {
	Slip  model.SlipGaji
	Admin bool
}

type RekapPage struct {
	Periode   []model.PilihanPeriode
	Baris     []model.RekapBaris
	Ringkasan model.RingkasanRekap
	IDPeriode int
	Jenis     string
}

func (h *KasbonHandler) Daftar(w http.ResponseWriter, r *http.Request) {
	daftar, err := h.Svc.KasbonDaftar(r.Context())
	if err != nil {
		gagal(w, err)
		return
	}
	u, _ := middleware.PenggunaDariContext(r.Context())
	h.View.Tampil(w, "kasbon/kasbon_list.html", KasbonDaftarPage{Daftar: daftar, Admin: u.AdalahAdmin()})
}

func (h *KasbonHandler) Form(w http.ResponseWriter, r *http.Request) {
	pekerja, err := h.Svc.PekerjaPilihan(r.Context())
	if err != nil {
		gagal(w, err)
		return
	}
	h.View.Tampil(w, "kasbon/kasbon_form.html", KasbonFormPage{Pekerja: pekerja})
}

func (h *KasbonHandler) renderKasbonFormError(w http.ResponseWriter, r *http.Request, page KasbonFormPage) {
	pekerja, err := h.Svc.PekerjaPilihan(r.Context())
	if err != nil {
		gagal(w, err)
		return
	}
	page.Pekerja = pekerja
	h.View.TampilStatus(w, http.StatusBadRequest, "kasbon/kasbon_form.html", page)
}

func (h *KasbonHandler) Simpan(w http.ResponseWriter, r *http.Request) {
	page := KasbonFormPage{
		IDPekerja:  strings.TrimSpace(r.PostFormValue("id_pekerja")),
		Tanggal:    strings.TrimSpace(r.PostFormValue("tanggal")),
		Jumlah:     strings.TrimSpace(r.PostFormValue("jumlah")),
		Keterangan: strings.TrimSpace(r.PostFormValue("keterangan")),
	}
	pekerjaID, err := strconv.Atoi(page.IDPekerja)
	if err != nil {
		page.Error = "Pekerja wajib dipilih"
		h.renderKasbonFormError(w, r, page)
		return
	}
	tanggal, err := time.Parse("2006-01-02", page.Tanggal)
	if err != nil {
		page.Error = "Tanggal pengambilan tidak valid"
		h.renderKasbonFormError(w, r, page)
		return
	}
	jumlah, err := strconv.ParseFloat(page.Jumlah, 64)
	if err != nil {
		page.Error = "Jumlah kasbon tidak valid"
		h.renderKasbonFormError(w, r, page)
		return
	}
	_, err = h.Svc.KasbonBuat(r.Context(), model.Kasbon{
		IDPekerja:  pekerjaID,
		Tanggal:    tanggal,
		Jumlah:     jumlah,
		Keterangan: page.Keterangan,
	})
	if err != nil {
		var ve *service.ValidasiError
		if errors.As(err, &ve) {
			page.Error = ve.Pesan
			h.renderKasbonFormError(w, r, page)
			return
		}
		gagal(w, err)
		return
	}
	http.Redirect(w, r, "/kasbon", http.StatusSeeOther)
}

func (h *KasbonHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	k, potongan, err := h.Svc.KasbonDetail(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}
	h.View.Tampil(w, "kasbon/kasbon_detail.html", KasbonDetailPage{Kasbon: k, Potongan: potongan})
}

func (h *KasbonHandler) Hapus(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	if err := h.Svc.KasbonHapus(r.Context(), id); err != nil {
		gagal(w, err)
		return
	}
	http.Redirect(w, r, "/kasbon", http.StatusSeeOther)
}

func (h *KasbonHandler) KoreksiDaftar(w http.ResponseWriter, r *http.Request) {
	daftar, err := h.Svc.KoreksiDaftar(r.Context())
	if err != nil {
		gagal(w, err)
		return
	}
	u, _ := middleware.PenggunaDariContext(r.Context())
	h.View.Tampil(w, "kasbon/koreksi_list.html", KoreksiDaftarPage{
		Daftar:  daftar,
		Admin:   u.AdalahAdmin(),
		Pemilik: u.AdalahPemilik(),
	})
}

func (h *KasbonHandler) KoreksiForm(w http.ResponseWriter, r *http.Request) {
	gaji, err := h.Svc.GajiDibayarPilihan(r.Context())
	if err != nil {
		gagal(w, err)
		return
	}
	h.View.Tampil(w, "kasbon/koreksi_form.html", KoreksiFormPage{Gaji: gaji})
}

func (h *KasbonHandler) renderKoreksiFormError(w http.ResponseWriter, r *http.Request, page KoreksiFormPage) {
	gaji, err := h.Svc.GajiDibayarPilihan(r.Context())
	if err != nil {
		gagal(w, err)
		return
	}
	page.Gaji = gaji
	h.View.TampilStatus(w, http.StatusBadRequest, "kasbon/koreksi_form.html", page)
}

func (h *KasbonHandler) KoreksiSimpan(w http.ResponseWriter, r *http.Request) {
	page := KoreksiFormPage{
		IDGajiAsal:   strings.TrimSpace(r.PostFormValue("id_gaji_asal")),
		Tanggal:      strings.TrimSpace(r.PostFormValue("tanggal")),
		NilaiSelisih: strings.TrimSpace(r.PostFormValue("nilai_selisih")),
		Alasan:       strings.TrimSpace(r.PostFormValue("alasan")),
	}
	idGaji, err := strconv.Atoi(page.IDGajiAsal)
	if err != nil {
		page.Error = "Gaji asal wajib dipilih"
		h.renderKoreksiFormError(w, r, page)
		return
	}
	tanggal, err := time.Parse("2006-01-02", page.Tanggal)
	if err != nil {
		page.Error = "Tanggal koreksi tidak valid"
		h.renderKoreksiFormError(w, r, page)
		return
	}
	selisih, err := strconv.ParseFloat(page.NilaiSelisih, 64)
	if err != nil {
		page.Error = "Nilai selisih tidak valid"
		h.renderKoreksiFormError(w, r, page)
		return
	}
	u, _ := middleware.PenggunaDariContext(r.Context())
	_, err = h.Svc.KoreksiBuat(r.Context(), model.KoreksiGaji{
		IDGajiAsal:   idGaji,
		DicatatOleh:  int(u.ID),
		Tanggal:      tanggal,
		NilaiSelisih: selisih,
		Alasan:       page.Alasan,
	})
	if err != nil {
		var ve *service.ValidasiError
		if errors.As(err, &ve) {
			page.Error = ve.Pesan
			h.renderKoreksiFormError(w, r, page)
			return
		}
		gagal(w, err)
		return
	}
	http.Redirect(w, r, "/koreksi", http.StatusSeeOther)
}

func (h *KasbonHandler) KoreksiSetujui(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	u, _ := middleware.PenggunaDariContext(r.Context())
	if err := h.Svc.KoreksiSetujui(r.Context(), id, u); err != nil {
		gagal(w, err)
		return
	}
	http.Redirect(w, r, "/koreksi", http.StatusSeeOther)
}

func (h *KasbonHandler) KoreksiHapus(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	if err := h.Svc.KoreksiHapus(r.Context(), id); err != nil {
		gagal(w, err)
		return
	}
	http.Redirect(w, r, "/koreksi", http.StatusSeeOther)
}

func (h *KasbonHandler) Slip(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	slip, err := h.Svc.Slip(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}
	u, _ := middleware.PenggunaDariContext(r.Context())
	h.View.Tampil(w, "kasbon/slip.html", SlipPage{Slip: slip, Admin: u.AdalahAdmin()})
}

func (h *KasbonHandler) HitungKomponenTambahan(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	if h.GajiSvc != nil {
		g, err := h.GajiSvc.Detail(r.Context(), id)
		if err != nil {
			gagal(w, err)
			return
		}
		if _, err := h.GajiSvc.HitungDraf(r.Context(), g.IDPekerja, g.IDPeriode, 0, 0); err != nil {
			gagal(w, err)
			return
		}
	} else if _, err := h.Svc.PotongOtomatis(r.Context(), id); err != nil {
		gagal(w, err)
		return
	}
	http.Redirect(w, r, "/gaji/"+strconv.Itoa(id)+"/slip", http.StatusSeeOther)
}

func (h *KasbonHandler) Rekap(w http.ResponseWriter, r *http.Request) {
	idPeriode := 0
	if raw := r.URL.Query().Get("periode"); raw != "" {
		var err error
		idPeriode, err = strconv.Atoi(raw)
		if err != nil || idPeriode <= 0 {
			http.Error(w, "Filter periode tidak valid", http.StatusBadRequest)
			return
		}
	}
	jenis := r.URL.Query().Get("jenis")
	baris, ringkasan, err := h.Svc.Rekap(r.Context(), idPeriode, jenis)
	if err != nil {
		gagal(w, err)
		return
	}
	periode, err := h.Svc.PeriodePilihan(r.Context())
	if err != nil {
		gagal(w, err)
		return
	}
	h.View.Tampil(w, "kasbon/rekap.html", RekapPage{
		Periode: periode, Baris: baris, Ringkasan: ringkasan, IDPeriode: idPeriode, Jenis: jenis,
	})
}
