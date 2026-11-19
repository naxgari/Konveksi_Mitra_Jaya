package service

import (
	"context"
	"time"

	"github.com/anggyputra/go-class/internal/model"
	"github.com/anggyputra/go-class/internal/repository"
)

type PeriodeGajiService struct {
	Repo *repository.PeriodeGajiRepo
}

func (s *PeriodeGajiService) Daftar(ctx context.Context) ([]model.PeriodeGaji, error) {
	return s.Repo.Semua(ctx)
}

func (s *PeriodeGajiService) Detail(ctx context.Context, id int) (model.PeriodeGaji, error) {
	if id <= 0 {
		return model.PeriodeGaji{}, &ValidasiError{
			Pesan: "ID periode tidak valid",
		}
	}

	return s.Repo.CariByID(ctx, id)
}

func (s *PeriodeGajiService) Buat(ctx context.Context, p model.PeriodeGaji) (int64, error) {
	// Tolak tanggal atau jenis yang salah sebelum mengakses database.
	if err := validasiPeriode(p, nil); err != nil {
		return 0, err
	}

	var id int64

	err := s.Repo.Transaksi(ctx, func(tx *repository.PeriodeGajiTx) error {
		daftar, err := tx.DaftarUntukValidasi(ctx)
		if err != nil {
			return err
		}

		if err := validasiPeriode(p, daftar); err != nil {
			return err
		}

		id, err = tx.Tambah(ctx, p)
		return err
	})
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (s *PeriodeGajiService) Tutup(
	ctx context.Context,
	penutup model.Pengguna,
	idPeriode int,
) error {
	if penutup.Peran != model.PeranPemilik {
		return &ValidasiError{Pesan: "Hanya Pemilik yang boleh menutup periode"}
	}
	if idPeriode <= 0 || penutup.ID <= 0 {
		return &ValidasiError{Pesan: "Periode dan pengguna wajib diisi"}
	}

	err := s.Repo.Transaksi(ctx, func(tx *repository.PeriodeGajiTx) error {
		status, err := tx.StatusPeriode(ctx, idPeriode)
		if err != nil {
			return err
		}

		daftar, err := tx.StatusGaji(ctx, idPeriode)
		if err != nil {
			return err
		}

		if err := validasiPenutupanPeriode(
			penutup.Peran, status, daftar,
		); err != nil {
			return err
		}

		return tx.Tutup(ctx, idPeriode, penutup.ID)
	})

	return errorAturan(err)
}

func validasiPeriodeMingguan(mulai, akhir, bayar time.Time) error {
	if mulai.IsZero() || akhir.IsZero() || bayar.IsZero() {
		return &ValidasiError{Pesan: "Tanggal periode dan pembayaran wajib diisi"}
	}

	sabtu := mulai.AddDate(0, 0, 5)
	if mulai.Weekday() != time.Monday ||
		akhir.Format("2006-01-02") != sabtu.Format("2006-01-02") {
		return &ValidasiError{Pesan: "Periode mingguan harus berjalan Senin sampai Sabtu"}
	}

	if bayar.Format("2006-01-02") != akhir.Format("2006-01-02") {
		return &ValidasiError{Pesan: "Pembayaran mingguan harus pada Sabtu akhir periode"}
	}

	return nil
}

func validasiPeriodeBulanan(mulai, akhir, bayar time.Time) error {
	if mulai.IsZero() || akhir.IsZero() || bayar.IsZero() {
		return &ValidasiError{Pesan: "Tanggal periode dan pembayaran wajib diisi"}
	}

	if mulai.Day() != 1 {
		return &ValidasiError{Pesan: "Periode bulanan harus dimulai tanggal 1"}
	}

	akhirBulan := mulai.AddDate(0, 1, -1)
	if akhir.Format("2006-01-02") != akhirBulan.Format("2006-01-02") {
		return &ValidasiError{Pesan: "Periode bulanan harus berakhir pada hari terakhir bulan"}
	}

	return nil
}

func hitungHariAktif(mulai, akhir, masuk, keluar time.Time) (int64, error) {
	if mulai.IsZero() || akhir.IsZero() || masuk.IsZero() {
		return 0, &ValidasiError{Pesan: "Tanggal periode dan masuk wajib diisi"}
	}

	const formatTanggal = "2006-01-02"

	awalPeriode := mulai.Format(formatTanggal)
	akhirPeriode := akhir.Format(formatTanggal)
	tanggalMasuk := masuk.Format(formatTanggal)

	if akhirPeriode < awalPeriode {
		return 0, &ValidasiError{Pesan: "Akhir periode tidak boleh sebelum awal periode"}
	}

	// Tanggal keluar kosong berarti pekerja masih bekerja.
	tanggalKeluar := ""
	if !keluar.IsZero() {
		tanggalKeluar = keluar.Format(formatTanggal)
		if tanggalKeluar < tanggalMasuk {
			return 0, &ValidasiError{Pesan: "Tanggal keluar tidak boleh sebelum tanggal masuk"}
		}
	}

	var jumlah int64

	for hari := mulai; hari.Format(formatTanggal) <= akhirPeriode; hari = hari.AddDate(0, 0, 1) {
		tanggal := hari.Format(formatTanggal)

		if tanggal < tanggalMasuk {
			continue
		}
		if tanggalKeluar != "" && tanggal > tanggalKeluar {
			continue
		}

		jumlah++
	}

	return jumlah, nil
}

func validasiPeriode(baru model.PeriodeGaji, daftar []model.PeriodeGaji) error {
	var err error

	switch baru.Jenis {
	case "mingguan":
		err = validasiPeriodeMingguan(
			baru.TanggalAwal,
			baru.TanggalAkhir,
			baru.TanggalBayarRencana,
		)
	case "bulanan":
		err = validasiPeriodeBulanan(
			baru.TanggalAwal,
			baru.TanggalAkhir,
			baru.TanggalBayarRencana,
		)
	default:
		return &ValidasiError{Pesan: "Jenis periode tidak valid"}
	}

	if err != nil {
		return err
	}

	const formatTanggal = "2006-01-02"
	awalBaru := baru.TanggalAwal.Format(formatTanggal)
	akhirBaru := baru.TanggalAkhir.Format(formatTanggal)

	for _, lama := range daftar {
		if lama.Jenis != baru.Jenis {
			continue
		}

		awalLama := lama.TanggalAwal.Format(formatTanggal)
		akhirLama := lama.TanggalAkhir.Format(formatTanggal)

		if awalBaru <= akhirLama && akhirBaru >= awalLama {
			return &ValidasiError{Pesan: "Periode sejenis tidak boleh bertumpang tindih"}
		}
	}

	return nil
}

func validasiPenutupanPeriode(
	peran model.PeranPengguna,
	statusPeriode string,
	daftarStatusGaji []string,
) error {
	if peran != model.PeranPemilik {
		return &ValidasiError{Pesan: "Hanya Pemilik yang boleh menutup periode"}
	}

	if statusPeriode != "terbuka" {
		return &ValidasiError{Pesan: "Hanya periode terbuka yang boleh ditutup"}
	}

	for _, status := range daftarStatusGaji {
		if status != "dibayar" {
			return &ValidasiError{Pesan: "Seluruh gaji harus sudah dibayar sebelum periode ditutup"}
		}
	}

	return nil
}
