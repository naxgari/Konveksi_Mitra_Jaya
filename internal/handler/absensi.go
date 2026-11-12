package handler

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/anggyputra/go-class/internal/middleware"
	"github.com/anggyputra/go-class/internal/model"
	"github.com/anggyputra/go-class/internal/service"
)

type AbsensiHandler struct {
	Svc  *service.AbsensiService
	View *Renderer
}
type JadwalPage struct {
	Admin                       bool
	Jadwal                      []model.JadwalShift
	Pekerja                     []model.PekerjaJadwal
	Mulai, Akhir, FilterPekerja string
	Error, Pesan                string
}
type JadwalFormPage struct {
	Judul, Subjudul, Action, FormPekerja, FormShift, Tanggal, Error string
	Pekerja                                                         []model.PekerjaJadwal
	Shift                                                           []model.Shift
}
type AbsensiFormPage struct {
	Judul, Action, Status, JamMasuk, JamKeluar, Keterangan, Error string
	Jadwal                                                        model.JadwalShift
}
type ShiftPage struct {
	Admin bool
	Shift []model.Shift
	Error string
}
type ShiftFormPage struct{ Judul, Action, Nama, JamMulai, JamSelesai, Error string }
type PemeriksaanPage struct {
	Admin                                                  bool
	Jadwal                                                 []model.JadwalShift
	Mulai, Akhir, FilterPekerja, FilterShift, FilterStatus string
	Pekerja                                                []model.PekerjaJadwal
	Shift                                                  []model.Shift
}

func (h *AbsensiHandler) JadwalDaftar(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.PenggunaDariContext(r.Context())
	if !user.AdalahAdmin() && !user.AdalahPemilik() {
		http.Error(w, "Akses ditolak", http.StatusForbidden)
		return
	}
	if err := validasiFilterTanggal(r.URL.Query()); err != nil {
		http.Error(w, "Filter tanggal tidak valid atau tanggal mulai melewati tanggal akhir", http.StatusBadRequest)
		return
	}
	js, err := h.Svc.JadwalDaftar(r.Context())
	if err != nil {
		gagal(w, err)
		return
	}
	ps, err := h.Svc.PekerjaDaftar(r.Context())
	if err != nil {
		gagal(w, err)
		return
	}
	q := r.URL.Query()
	p := JadwalPage{Admin: user.AdalahAdmin(), Pekerja: ps, Mulai: q.Get("mulai"), Akhir: q.Get("akhir"), FilterPekerja: q.Get("pekerja")}
	for _, j := range js {
		if !dalamFilter(j, q) || !filterPekerja(j, q) {
			continue
		}
		p.Jadwal = append(p.Jadwal, j)
	}
	h.View.Tampil(w, "absensi/jadwal_list.html", p)
}
func (h *AbsensiHandler) JadwalFormTambah(w http.ResponseWriter, r *http.Request) {
	h.jadwalForm(w, r, model.JadwalShift{}, "")
}
func (h *AbsensiHandler) JadwalFormEdit(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	j, err := h.Svc.JadwalDetail(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}
	h.jadwalForm(w, r, j, "")
}
func (h *AbsensiHandler) jadwalForm(w http.ResponseWriter, r *http.Request, j model.JadwalShift, msg string) {
	pekerja := ""
	shift := ""
	if j.IDPekerja > 0 {
		pekerja = strconv.Itoa(j.IDPekerja)
	}
	if j.IDShift > 0 {
		shift = strconv.Itoa(j.IDShift)
	}
	tanggal := ""
	if !j.Tanggal.IsZero() {
		tanggal = j.Tanggal.Format("2006-01-02")
	}
	h.jadwalFormValues(w, r, j, msg, pekerja, shift, tanggal)
}

func (h *AbsensiHandler) jadwalFormValues(w http.ResponseWriter, r *http.Request, j model.JadwalShift, msg, pekerja, shift, tanggal string) {
	ps, e := h.Svc.PekerjaDaftar(r.Context())
	if e != nil {
		gagal(w, e)
		return
	}
	ss, e := h.Svc.ShiftDaftar(r.Context())
	if e != nil {
		gagal(w, e)
		return
	}
	act := "/jadwal"
	title := "Tempatkan Shift"
	if j.ID > 0 {
		act = "/jadwal/" + strconv.Itoa(j.ID) + "/edit"
		title = "Edit Jadwal"
	}
	h.View.Tampil(w, "absensi/jadwal_form.html", JadwalFormPage{Judul: title, Subjudul: "Jadwal kerja", Action: act, FormPekerja: pekerja, FormShift: shift, Tanggal: tanggal, Pekerja: ps, Shift: ss, Error: msg})
}
func (h *AbsensiHandler) JadwalSimpan(w http.ResponseWriter, r *http.Request) {
	h.jadwalMutasi(w, r, false)
}
func (h *AbsensiHandler) JadwalUpdate(w http.ResponseWriter, r *http.Request) {
	h.jadwalMutasi(w, r, true)
}
func (h *AbsensiHandler) jadwalMutasi(w http.ResponseWriter, r *http.Request, edit bool) {
	var j model.JadwalShift
	if edit {
		id, ok := idDariPath(w, r)
		if !ok {
			return
		}
		j.ID = id
	}
	pekerja := r.PostFormValue("pekerja")
	shift := r.PostFormValue("shift")
	tanggal := r.PostFormValue("tanggal")
	var parseErr error
	j.IDPekerja, parseErr = strconv.Atoi(pekerja)
	if parseErr != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.jadwalFormValues(w, r, j, "Pilih pekerja yang valid", pekerja, shift, tanggal)
		return
	}
	j.IDShift, parseErr = strconv.Atoi(shift)
	if parseErr != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.jadwalFormValues(w, r, j, "Pilih shift yang valid", pekerja, shift, tanggal)
		return
	}
	parsedDate, dateErr := time.Parse("2006-01-02", tanggal)
	j.Tanggal = parsedDate
	if dateErr != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.jadwalFormValues(w, r, j, "Tanggal tidak valid", pekerja, shift, tanggal)
		return
	}
	var err error
	if edit {
		err = h.Svc.JadwalUbah(r.Context(), j)
	} else {
		_, err = h.Svc.JadwalBuat(r.Context(), j)
	}
	if err != nil {
		if validasi(err) {
			w.WriteHeader(http.StatusBadRequest)
			h.jadwalFormValues(w, r, j, validasiPesan(err), pekerja, shift, tanggal)
			return
		}
		gagal(w, err)
		return
	}
	http.Redirect(w, r, "/jadwal", http.StatusSeeOther)
}
func (h *AbsensiHandler) JadwalHapus(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	if err := h.Svc.JadwalHapus(r.Context(), id); err != nil {
		gagal(w, err)
		return
	}
	http.Redirect(w, r, "/jadwal", http.StatusSeeOther)
}
func (h *AbsensiHandler) AbsensiForm(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	j, e := h.Svc.JadwalDetail(r.Context(), id)
	if e != nil {
		gagal(w, e)
		return
	}
	p := AbsensiFormPage{Judul: "Catat Absensi", Action: "/jadwal/" + strconv.Itoa(id) + "/absensi", Jadwal: j, Status: "perlu_diperiksa"}
	if j.Absensi != nil {
		p.Status = j.Absensi.Status
		p.Keterangan = j.Absensi.Keterangan
		if j.Absensi.JamMasuk != nil {
			p.JamMasuk = j.Absensi.JamMasuk.Format("15:04")
		}
		if j.Absensi.JamKeluar != nil {
			p.JamKeluar = j.Absensi.JamKeluar.Format("15:04")
		}
	}
	h.View.Tampil(w, "absensi/absensi_form.html", p)
}
func (h *AbsensiHandler) AbsensiSimpan(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	j, e := h.Svc.JadwalDetail(r.Context(), id)
	if e != nil {
		gagal(w, e)
		return
	}
	a := model.Absensi{IDJadwal: id, Status: r.PostFormValue("status"), Keterangan: strings.TrimSpace(r.PostFormValue("keterangan"))}
	jamMasukInput := r.PostFormValue("jam_masuk")
	jamKeluarInput := r.PostFormValue("jam_keluar")
	for _, item := range []struct {
		value string
		dst   **time.Time
	}{{jamMasukInput, &a.JamMasuk}, {jamKeluarInput, &a.JamKeluar}} {
		v := item.value
		if v == "" {
			continue
		}
		t, err := time.ParseInLocation("2006-01-02 15:04", j.Tanggal.Format("2006-01-02")+" "+v, time.Local)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			h.absensiFormError(w, r, j, a, "Format jam tidak valid", jamMasukInput, jamKeluarInput)
			return
		}
		*item.dst = &t
	}
	if err := h.Svc.AbsensiSimpan(r.Context(), a); err != nil {
		if validasi(err) {
			w.WriteHeader(http.StatusBadRequest)
			h.absensiFormError(w, r, j, a, validasiPesan(err), jamMasukInput, jamKeluarInput)
			return
		}
		gagal(w, err)
		return
	}
	http.Redirect(w, r, "/absensi/pemeriksaan", http.StatusSeeOther)
}
func (h *AbsensiHandler) absensiFormError(w http.ResponseWriter, r *http.Request, j model.JadwalShift, a model.Absensi, msg, jamMasuk, jamKeluar string) {
	p := AbsensiFormPage{Judul: "Catat Absensi", Action: "/jadwal/" + strconv.Itoa(j.ID) + "/absensi", Jadwal: j, Status: a.Status, Keterangan: a.Keterangan, Error: msg, JamMasuk: jamMasuk, JamKeluar: jamKeluar}
	h.View.Tampil(w, "absensi/absensi_form.html", p)
}
func (h *AbsensiHandler) Pemeriksaan(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.PenggunaDariContext(r.Context())
	if !user.AdalahAdmin() && !user.AdalahPemilik() {
		http.Error(w, "Akses ditolak", http.StatusForbidden)
		return
	}
	if err := validasiFilterTanggal(r.URL.Query()); err != nil {
		http.Error(w, "Filter tanggal tidak valid atau tanggal mulai melewati tanggal akhir", http.StatusBadRequest)
		return
	}
	js, e := h.Svc.JadwalDaftar(r.Context())
	if e != nil {
		gagal(w, e)
		return
	}
	ps, e := h.Svc.PekerjaDaftar(r.Context())
	if e != nil {
		gagal(w, e)
		return
	}
	ss, e := h.Svc.ShiftDaftar(r.Context())
	if e != nil {
		gagal(w, e)
		return
	}
	q := r.URL.Query()
	p := PemeriksaanPage{Admin: user.AdalahAdmin(), Mulai: q.Get("mulai"), Akhir: q.Get("akhir"), FilterPekerja: q.Get("pekerja"), FilterShift: q.Get("shift"), FilterStatus: q.Get("status"), Pekerja: ps, Shift: ss}
	for _, j := range js {
		if !dalamFilter(j, q) || !filterPekerja(j, q) || !filterShift(j, q) {
			continue
		}
		status := statusAbsensi(j)
		if p.FilterStatus != "" && status != p.FilterStatus {
			continue
		}
		p.Jadwal = append(p.Jadwal, j)
	}
	h.View.Tampil(w, "absensi/pemeriksaan.html", p)
}
func statusAbsensi(j model.JadwalShift) string {
	if j.Absensi == nil {
		return "belum_dicatat"
	}
	a := j.Absensi
	if a.Status == "tidak_hadir" {
		return "tidak_hadir"
	}
	if a.Status != "hadir" || a.JamMasuk == nil || a.JamKeluar == nil {
		return "perlu_diperiksa"
	}
	return "hadir"
}
func dalamFilter(j model.JadwalShift, q url.Values) bool {
	from := q.Get("mulai")
	to := q.Get("akhir")
	d := j.Tanggal.Format("2006-01-02")
	return (from == "" || d >= from) && (to == "" || d <= to)
}
func filterPekerja(j model.JadwalShift, q url.Values) bool {
	return q.Get("pekerja") == "" || strconv.Itoa(j.IDPekerja) == q.Get("pekerja")
}
func filterShift(j model.JadwalShift, q url.Values) bool {
	return q.Get("shift") == "" || strconv.Itoa(j.IDShift) == q.Get("shift")
}
func validasiFilterTanggal(q url.Values) error {
	var mulai, akhir time.Time
	var err error
	if v := q.Get("mulai"); v != "" {
		mulai, err = time.Parse("2006-01-02", v)
		if err != nil {
			return err
		}
	}
	if v := q.Get("akhir"); v != "" {
		akhir, err = time.Parse("2006-01-02", v)
		if err != nil {
			return err
		}
	}
	if !mulai.IsZero() && !akhir.IsZero() && mulai.After(akhir) {
		return errors.New("rentang tanggal terbalik")
	}
	return nil
}
func validasi(err error) bool { var ve *service.ValidasiError; return errors.As(err, &ve) }
func validasiPesan(err error) string {
	var ve *service.ValidasiError
	if errors.As(err, &ve) {
		return ve.Pesan
	}
	return "Input belum valid"
}

func (h *AbsensiHandler) ShiftDaftar(w http.ResponseWriter, r *http.Request) {
	u, _ := middleware.PenggunaDariContext(r.Context())
	if !u.AdalahAdmin() && !u.AdalahPemilik() {
		http.Error(w, "Akses ditolak", http.StatusForbidden)
		return
	}
	ss, e := h.Svc.ShiftDaftar(r.Context())
	if e != nil {
		gagal(w, e)
		return
	}
	h.View.Tampil(w, "absensi/shift_list.html", ShiftPage{Admin: u.AdalahAdmin(), Shift: ss})
}
func (h *AbsensiHandler) ShiftFormTambah(w http.ResponseWriter, r *http.Request) {
	h.shiftForm(w, r, model.Shift{}, "")
}
func (h *AbsensiHandler) ShiftFormEdit(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	s, e := h.Svc.ShiftDetail(r.Context(), id)
	if e != nil {
		gagal(w, e)
		return
	}
	h.shiftForm(w, r, s, "")
}
func (h *AbsensiHandler) shiftForm(w http.ResponseWriter, r *http.Request, s model.Shift, msg string) {
	p := ShiftFormPage{Judul: "Tambah Shift", Action: "/shift", Nama: s.Nama, JamMulai: s.JamMulai, JamSelesai: s.JamSelesai, Error: msg}
	if s.ID > 0 {
		p.Judul = "Edit Shift"
		p.Action = "/shift/" + strconv.Itoa(s.ID) + "/edit"
	}
	h.View.Tampil(w, "absensi/shift_form.html", p)
}
func (h *AbsensiHandler) ShiftSimpan(w http.ResponseWriter, r *http.Request) {
	h.shiftMutasi(w, r, false)
}
func (h *AbsensiHandler) ShiftUpdate(w http.ResponseWriter, r *http.Request) {
	h.shiftMutasi(w, r, true)
}
func (h *AbsensiHandler) shiftMutasi(w http.ResponseWriter, r *http.Request, edit bool) {
	s := model.Shift{Nama: strings.TrimSpace(r.PostFormValue("nama")), JamMulai: r.PostFormValue("mulai"), JamSelesai: r.PostFormValue("selesai")}
	if edit {
		id, ok := idDariPath(w, r)
		if !ok {
			return
		}
		s.ID = id
	}
	var e error
	if edit {
		e = h.Svc.ShiftUbah(r.Context(), s)
	} else {
		_, e = h.Svc.ShiftBuat(r.Context(), s)
	}
	if e != nil {
		if validasi(e) {
			w.WriteHeader(http.StatusBadRequest)
			h.shiftForm(w, r, s, validasiPesan(e))
			return
		}
		gagal(w, e)
		return
	}
	http.Redirect(w, r, "/shift", http.StatusSeeOther)
}
func (h *AbsensiHandler) ShiftHapus(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	if e := h.Svc.ShiftHapus(r.Context(), id); e != nil {
		gagal(w, e)
		return
	}
	http.Redirect(w, r, "/shift", http.StatusSeeOther)
}
