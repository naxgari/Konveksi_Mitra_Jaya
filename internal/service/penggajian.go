package service

import (
	"context"
	"time"

	"github.com/anggyputra/go-class/internal/model"
	"github.com/anggyputra/go-class/internal/repository"
)

type GajiService struct {
	Repo *repository.GajiRepo
}


func (s *GajiService) DaftarPerPeriode(
	ctx context.Context,
	idPeriode int,
) ([]model.Gaji, error) {
	if idPeriode <= 0 {
		return nil, &ValidasiError{Pesan: "ID periode tidak valid"}
	}

	repoPeriode := &repository.PeriodeGajiRepo{DB: s.Repo.DB}

	if _, err := repoPeriode.CariByID(ctx, idPeriode); err != nil {
		return nil, err
	}

	return s.Repo.DaftarPerPeriode(ctx, idPeriode)
}

func (s *GajiService) Detail(
	ctx context.Context,
	idGaji int,
) (model.Gaji, error) {
	if idGaji <= 0 {
		return model.Gaji{}, &ValidasiError{Pesan: "ID gaji tidak valid"}
	}

	return s.Repo.CariByID(ctx, idGaji)
}

func (s *GajiService) Rincian(
	ctx context.Context,
	idGaji int,
) ([]model.DetailGaji, error) {
	if _, err := s.Detail(ctx, idGaji); err != nil {
		return nil, err
	}

	return s.Repo.DetailPerGaji(ctx, idGaji)
}



func (s *GajiService) Bayar(
	ctx context.Context,
	peran model.PeranPengguna,
	idPekerja, idPeriode int,
) error {
	if peran != model.PeranAdmin {
		return &ValidasiError{Pesan: "Pembayaran hanya boleh dilakukan Admin"}
	}
	if idPekerja <= 0 || idPeriode <= 0 {
		return &ValidasiError{Pesan: "Pekerja dan periode wajib diisi"}
	}

	err := s.Repo.Transaksi(ctx, func(tx *repository.GajiTx) error {
		pekerja, err := tx.Pekerja(ctx, idPekerja)
		if err != nil {
			return err
		}

		periode, err := tx.Periode(ctx, idPeriode)
		if err != nil {
			return err
		}

		gaji, err := tx.GajiPekerja(ctx, idPekerja, idPeriode)
		if err != nil {
			return err
		}
		if err := validasiPembayaran(periode.Status, gaji.StatusBayar); err != nil {
			return err
		}
		if err := validasiPeriode(periode, nil); err != nil {
			return err
		}

		jenisSesuai :=
			(pekerja.Jenis == "harian" && periode.Jenis == "mingguan") ||
				(pekerja.Jenis == "bulanan" && periode.Jenis == "bulanan")
		if !jenisSesuai {
			return &ValidasiError{Pesan: "Jenis pekerja tidak sesuai dengan periode"}
		}

		keluar := time.Time{}
		if pekerja.Keluar.Valid {
			keluar = pekerja.Keluar.Time
		}
		aktif, err := hitungHariAktif(
			periode.TanggalAwal, periode.TanggalAkhir,
			pekerja.Masuk, keluar,
		)
		if err != nil {
			return err
		}
		if aktif == 0 {
			return &ValidasiError{Pesan: "Pekerja tidak aktif pada periode ini"}
		}

		if gaji.TotalPotonganKasbon != "0.00" || gaji.TotalKoreksi != "0.00" {
			return &ValidasiError{
				Pesan: "Potongan kasbon atau koreksi perlu diperiksa sebelum pembayaran",
			}
		}

		perlu, err := tx.PerluPenyesuaianPembayaran(
			ctx, idPekerja, gaji.ID, periode,
		)
		if err != nil {
			return err
		}
		if perlu {
			return &ValidasiError{
				Pesan: "Potongan kasbon atau koreksi perlu diperiksa sebelum pembayaran",
			}
		}

		// Periksa kembali perhitungan berdasarkan data terbaru.
		total, rincian, err := hitungRincianDraf(
			ctx, tx, idPekerja, periode, pekerja,
		)
		if err != nil {
			return err
		}
		hasil, err := hitungTotalGaji(total, 0, 0)
		if err != nil {
			return err
		}
		if hasil.Bersih.Sign() < 0 {
			return &ValidasiError{Pesan: "Gaji negatif perlu diperiksa sebelum dibayar"}
		}

		if gaji.GajiPokok != total.Pokok.FloatString(2) ||
			gaji.TotalLembur != total.Lembur.FloatString(2) ||
			gaji.PotonganTelat != total.PotonganTelat.FloatString(2) ||
			gaji.GajiBruto != hasil.Bruto.FloatString(2) ||
			gaji.GajiBersih != hasil.Bersih.FloatString(0)+".00" {
			return &ValidasiError{
				Pesan: "Data penggajian berubah. Hitung ulang draf sebelum membayar",
			}
		}

		// Simpan rincian terbaru sebelum mengunci hasil lewat pembayaran.
		if _, err := simpanDraf(ctx, tx, gaji, rincian); err != nil {
			return err
		}

		return tx.CatatPembayaran(ctx, gaji.ID)
	})

	return errorAturan(err)
}

func validasiPembayaran(statusPeriode, statusBayar string) error {
	if statusPeriode != "terbuka" {
		return &ValidasiError{Pesan: "Pembayaran hanya boleh pada periode terbuka"}
	}

	if statusBayar == "dibayar" {
		return &ValidasiError{Pesan: "Gaji sudah dibayar"}
	}

	if statusBayar != "belum" {
		return &ValidasiError{Pesan: "Status pembayaran tidak valid"}
	}

	return nil
}

func (s *GajiService) PekerjaPerPeriode(
	ctx context.Context,
	idPeriode int,
) ([]model.PekerjaGaji, error) {
	if idPeriode <= 0 {
		return nil, &ValidasiError{Pesan: "ID periode tidak valid"}
	}

	return s.Repo.PekerjaPerPeriode(ctx, idPeriode)
}