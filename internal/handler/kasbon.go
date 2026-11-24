package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/anggyputra/go-class/internal/middleware"
	"github.com/anggyputra/go-class/internal/model"
	"github.com/anggyputra/go-class/internal/service"
)

type KasbonHandler struct {
	Svc  *service.KasbonService
	View *Renderer
}

type KasbonDaftarPage struct {
	Daftar []model.Kasbon
	Admin  bool
}

type KasbonFormPage struct {
	Pekerja []model.PekerjaJadwal
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
	Gaji []model.PilihanGaji
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

func (h *KasbonHandler) Simpan(w http.ResponseWriter, r *http.Request) {
	pekerjaID, err := strconv.Atoi(r.PostFormValue("id_pekerja"))
	if err != nil {
		http.Error(w, "Pekerja tidak valid", http.StatusBadRequest)
		return
	}
	tanggal, err := time.Parse("2006-01-02", r.PostFormValue("tanggal"))
	if err != nil {
		http.Error(w, "Tanggal tidak valid", http.StatusBadRequest)
		return
	}
	jumlah, err := strconv.ParseFloat(r.PostFormValue("jumlah"), 64)
	if err != nil {
		http.Error(w, "Jumlah kasbon tidak valid", http.StatusBadRequest)
		return
	}
	_, err = h.Svc.KasbonBuat(r.Context(), model.Kasbon{
		IDPekerja:  pekerjaID,
		Tanggal:    tanggal,
		Jumlah:     jumlah,
		Keterangan: r.PostFormValue("keterangan"),
	})
	if err != nil {
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

func (h *KasbonHandler) KoreksiSimpan(w http.ResponseWriter, r *http.Request) {
	idGaji, err := strconv.Atoi(r.PostFormValue("id_gaji_asal"))
	if err != nil {
		http.Error(w, "Gaji asal tidak valid", http.StatusBadRequest)
		return
	}
	tanggal, err := time.Parse("2006-01-02", r.PostFormValue("tanggal"))
	if err != nil {
		http.Error(w, "Tanggal tidak valid", http.StatusBadRequest)
		return
	}
	selisih, err := strconv.ParseFloat(r.PostFormValue("nilai_selisih"), 64)
	if err != nil {
		http.Error(w, "Nilai selisih tidak valid", http.StatusBadRequest)
		return
	}
	u, _ := middleware.PenggunaDariContext(r.Context())
	_, err = h.Svc.KoreksiBuat(r.Context(), model.KoreksiGaji{
		IDGajiAsal:   idGaji,
		DicatatOleh:  int(u.ID),
		Tanggal:      tanggal,
		NilaiSelisih: selisih,
		Alasan:       r.PostFormValue("alasan"),
	})
	if err != nil {
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
	if err := h.Svc.KoreksiSetujui(r.Context(), id, int(u.ID)); err != nil {
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
	h.View.Tampil(w, "kasbon/slip.html", slip)
}

func (h *KasbonHandler) HitungKomponenTambahan(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	if _, err := h.Svc.PotongOtomatis(r.Context(), id); err != nil {
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
