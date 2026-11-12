package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/anggyputra/go-class/internal/model"
	"github.com/anggyputra/go-class/internal/repository"
)

type AbsensiService struct{ Repo *repository.AbsensiRepo }

func (s *AbsensiService) ShiftDaftar(ctx context.Context) ([]model.Shift, error) {
	return s.Repo.ShiftDaftar(ctx)
}
func (s *AbsensiService) ShiftDetail(ctx context.Context, id int) (model.Shift, error) {
	return s.Repo.ShiftDetail(ctx, id)
}
func (s *AbsensiService) PekerjaDaftar(ctx context.Context) ([]model.PekerjaJadwal, error) {
	return s.Repo.PekerjaDaftar(ctx)
}
func (s *AbsensiService) JadwalDaftar(ctx context.Context) ([]model.JadwalShift, error) {
	return s.Repo.JadwalDaftar(ctx)
}
func (s *AbsensiService) JadwalDetail(ctx context.Context, id int) (model.JadwalShift, error) {
	return s.Repo.JadwalDetail(ctx, id)
}
func validasiShift(v model.Shift) error {
	if strings.TrimSpace(v.Nama) == "" || utf8.RuneCountInString(v.Nama) > 50 {
		return &ValidasiError{"Nama shift wajib diisi dan maksimal 50 karakter"}
	}
	a, e := time.Parse("15:04", v.JamMulai)
	if e != nil {
		return &ValidasiError{"Jam mulai tidak valid"}
	}
	b, e := time.Parse("15:04", v.JamSelesai)
	if e != nil || !b.After(a) {
		return &ValidasiError{"Jam selesai harus setelah jam mulai pada hari yang sama"}
	}
	return nil
}
func (s *AbsensiService) ShiftBuat(ctx context.Context, v model.Shift) (int64, error) {
	if err := validasiShift(v); err != nil {
		return 0, err
	}
	id, err := s.Repo.ShiftBuat(ctx, v)
	return id, errorAturan(err)
}
func (s *AbsensiService) ShiftUbah(ctx context.Context, v model.Shift) error {
	if err := validasiShift(v); err != nil {
		return err
	}
	return s.shiftMutasi(ctx, v, false)
}
func (s *AbsensiService) ShiftHapus(ctx context.Context, id int) error {
	return s.shiftMutasi(ctx, model.Shift{ID: id}, true)
}
func (s *AbsensiService) shiftMutasi(ctx context.Context, v model.Shift, hapus bool) error {
	return errorAturan(s.Repo.Transaksi(ctx, func(tx *repository.AbsensiTx) error {
		if _, e := tx.Shift(ctx, v.ID); e != nil {
			return e
		}
		n, e := tx.JumlahJadwalShift(ctx, v.ID)
		if e != nil {
			return e
		}
		if n > 0 {
			return &ValidasiError{"Shift yang telah digunakan jadwal tidak dapat diubah atau dihapus"}
		}
		if hapus {
			return tx.HapusShift(ctx, v.ID)
		}
		return tx.UbahShift(ctx, v)
	}))
}
func validasiJadwal(v model.JadwalShift) error {
	if v.IDPekerja <= 0 || v.IDShift <= 0 || v.Tanggal.IsZero() {
		return &ValidasiError{"Pekerja, shift, dan tanggal wajib diisi"}
	}
	return nil
}
func periksaPekerja(ctx context.Context, tx *repository.AbsensiTx, id int, tanggal time.Time) error {
	p, e := tx.Pekerja(ctx, id)
	if e != nil {
		return e
	}
	tgl := tanggal.Format("2006-01-02")
	if tgl < p.Masuk.Format("2006-01-02") || (p.Keluar.Valid && tgl > p.Keluar.Time.Format("2006-01-02")) {
		return &ValidasiError{"Tanggal jadwal di luar masa aktif pekerja"}
	}
	jenis := "mingguan"
	if p.Jenis == "bulanan" {
		jenis = "bulanan"
	}
	periode, e := tx.Periode(ctx, id, jenis, tgl)
	if e != nil {
		return e
	}
	for _, p := range periode {
		if p.Status == "tertutup" {
			return &ValidasiError{"Periode gaji sudah ditutup"}
		}
		if p.StatusBayar == "dibayar" {
			return &ValidasiError{"Gaji pekerja pada periode ini sudah dibayar"}
		}
	}
	return nil
}
func (s *AbsensiService) JadwalBuat(ctx context.Context, v model.JadwalShift) (int64, error) {
	if e := validasiJadwal(v); e != nil {
		return 0, e
	}
	var id int64
	e := s.Repo.Transaksi(ctx, func(tx *repository.AbsensiTx) error {
		if e := periksaPekerja(ctx, tx, v.IDPekerja, v.Tanggal); e != nil {
			return e
		}
		if _, e := tx.Shift(ctx, v.IDShift); e != nil {
			return e
		}
		var e error
		id, e = tx.TambahJadwal(ctx, v)
		return e
	})
	return id, errorAturan(e)
}
func (s *AbsensiService) JadwalUbah(ctx context.Context, v model.JadwalShift) error {
	if e := validasiJadwal(v); e != nil {
		return e
	}
	return s.jadwalMutasi(ctx, v, false)
}
func (s *AbsensiService) JadwalHapus(ctx context.Context, id int) error {
	return s.jadwalMutasi(ctx, model.JadwalShift{ID: id}, true)
}
func (s *AbsensiService) jadwalMutasi(ctx context.Context, v model.JadwalShift, hapus bool) error {
	return errorAturan(s.Repo.Transaksi(ctx, func(tx *repository.AbsensiTx) error {
		old, e := tx.Jadwal(ctx, v.ID)
		if e != nil {
			return e
		}
		if e = periksaPekerja(ctx, tx, old.IDPekerja, old.Tanggal); e != nil {
			return e
		}
		if old.Absensi != nil {
			return &ValidasiError{"Jadwal yang sudah memiliki absensi tidak dapat diubah atau dihapus"}
		}
		if hapus {
			return tx.HapusJadwal(ctx, v.ID)
		}
		if e = periksaPekerja(ctx, tx, v.IDPekerja, v.Tanggal); e != nil {
			return e
		}
		if _, e = tx.Shift(ctx, v.IDShift); e != nil {
			return e
		}
		return tx.UbahJadwal(ctx, v)
	}))
}
func hitungAbsensi(a *model.Absensi, j model.JadwalShift) error {
	if utf8.RuneCountInString(a.Keterangan) > 255 {
		return &ValidasiError{"Keterangan maksimal 255 karakter"}
	}
	if a.Status != "hadir" && a.Status != "tidak_hadir" && a.Status != "perlu_diperiksa" {
		return &ValidasiError{"Status kehadiran tidak valid"}
	}
	if a.Status == "hadir" && (a.JamMasuk == nil || a.JamKeluar == nil) {
		return &ValidasiError{"Hadir memerlukan jam masuk dan keluar"}
	}
	if a.Status == "tidak_hadir" && (a.JamMasuk != nil || a.JamKeluar != nil) {
		return &ValidasiError{"Tidak hadir tidak memiliki jam masuk atau keluar"}
	}
	for _, t := range []*time.Time{a.JamMasuk, a.JamKeluar} {
		if t != nil && t.Format("2006-01-02") != j.Tanggal.Format("2006-01-02") {
			return &ValidasiError{"Tanggal jam harus sama dengan tanggal jadwal"}
		}
	}
	if a.JamMasuk != nil && a.JamKeluar != nil && a.JamKeluar.Before(*a.JamMasuk) {
		return &ValidasiError{"Jam keluar tidak boleh sebelum jam masuk"}
	}
	a.MenitTerlambat = 0
	a.MenitLembur = 0
	if a.Status == "hadir" {
		start, e := time.ParseInLocation("2006-01-02 15:04", j.Tanggal.Format("2006-01-02")+" "+j.JamMulai, a.JamMasuk.Location())
		if e != nil {
			return e
		}
		end, e := time.ParseInLocation("2006-01-02 15:04", j.Tanggal.Format("2006-01-02")+" "+j.JamSelesai, a.JamKeluar.Location())
		if e != nil {
			return e
		}
		if a.JamMasuk.After(start) {
			a.MenitTerlambat = int(a.JamMasuk.Sub(start) / time.Minute)
		}
		if a.JamKeluar.After(end) {
			a.MenitLembur = int(a.JamKeluar.Sub(end) / time.Minute)
		}
	}
	return nil
}
func (s *AbsensiService) AbsensiSimpan(ctx context.Context, a model.Absensi) error {
	return s.Repo.Transaksi(ctx, func(tx *repository.AbsensiTx) error {
		j, e := tx.Jadwal(ctx, a.IDJadwal)
		if e != nil {
			return e
		}
		if e = periksaPekerja(ctx, tx, j.IDPekerja, j.Tanggal); e != nil {
			return e
		}
		if e = hitungAbsensi(&a, j); e != nil {
			return e
		}
		return tx.SimpanAbsensi(ctx, a)
	})
}
func errorAturan(err error) error {
	var e *repository.AturanError
	if errors.As(err, &e) {
		return &ValidasiError{Pesan: e.Pesan}
	}
	return err
}
