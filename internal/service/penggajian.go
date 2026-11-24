package service

import (
	"context"
	"math/big"
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

		koreksiList, err := tx.KoreksiDisetujui(ctx, idPekerja, periode)
		if err != nil {
			return err
		}
		var koreksiSen int64
		koreksiIDs := make([]int, 0, len(koreksiList))
		for _, k := range koreksiList {
			koreksiSen += k.NilaiSen
			koreksiIDs = append(koreksiIDs, k.ID)
		}

		if aktif == 0 && len(koreksiList) == 0 {
			return &ValidasiError{Pesan: "Pekerja tidak aktif pada periode ini"}
		}

		// Periksa kembali perhitungan, saldo kasbon, dan koreksi berdasarkan data terbaru.
		total := &rincianGaji{}
		var rincian []model.DetailGaji
		if aktif > 0 {
			total, rincian, err = hitungRincianDraf(
				ctx, tx, idPekerja, periode, pekerja,
			)
			if err != nil {
				return err
			}
		}

		bruto := new(big.Rat).Add(&total.Pokok, &total.Lembur)
		daftarKasbon, err := tx.KasbonBersaldo(ctx, idPekerja, periode.TanggalAkhir)
		if err != nil {
			return err
		}
		alokasi, totalKasbonSen, sebelumNegatif := hitungAlokasiKasbon(
			bruto, &total.PotonganTelat, koreksiSen, daftarKasbon,
		)
		if sebelumNegatif {
			return &ValidasiError{Pesan: "Nilai gaji sebelum potongan kasbon negatif; pembayaran ditahan untuk pemeriksaan"}
		}

		hasil, err := hitungTotalGaji(total, totalKasbonSen, koreksiSen)
		if err != nil {
			return err
		}
		if hasil.Bersih.Sign() < 0 {
			return &ValidasiError{Pesan: "Gaji negatif perlu diperiksa sebelum dibayar"}
		}

		if gaji.GajiPokok != total.Pokok.FloatString(2) ||
			gaji.TotalLembur != total.Lembur.FloatString(2) ||
			gaji.PotonganTelat != total.PotonganTelat.FloatString(2) ||
			gaji.TotalPotonganKasbon != big.NewRat(totalKasbonSen, 100).FloatString(2) ||
			gaji.TotalKoreksi != big.NewRat(koreksiSen, 100).FloatString(2) ||
			gaji.GajiBruto != hasil.Bruto.FloatString(2) ||
			gaji.GajiBersih != hasil.Bersih.FloatString(0)+".00" {
			return &ValidasiError{
				Pesan: "Data penggajian, kasbon, atau koreksi berubah. Hitung ulang draf sebelum membayar",
			}
		}

		// Simpan rincian, potongan kasbon final, koreksi diterapkan, dan pembayaran dalam satu transaksi atomik.
		idGaji, err := simpanDraf(ctx, tx, gaji, rincian)
		if err != nil {
			return err
		}
		if err := tx.HapusPotonganKasbonDraf(ctx, int(idGaji)); err != nil {
			return err
		}
		for _, a := range alokasi {
			if err := tx.TambahPotonganKasbon(
				ctx, int(idGaji), a.IDKasbon, big.NewRat(a.JumlahSen, 100).FloatString(2),
			); err != nil {
				return err
			}
		}
		if err := tx.TerapkanKoreksi(ctx, int(idGaji), koreksiIDs); err != nil {
			return err
		}

		return tx.CatatPembayaran(ctx, int(idGaji))
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