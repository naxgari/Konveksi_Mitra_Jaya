package service

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/anggyputra/go-class/internal/model"
	"github.com/anggyputra/go-class/internal/repository"
)

func (s *GajiService) HitungDraf(
	ctx context.Context,
	idPekerja, idPeriode int,
	kasbonSen, koreksiSen int64,
) (int64, error) {
	if idPekerja <= 0 || idPeriode <= 0 {
		return 0, &ValidasiError{Pesan: "Pekerja dan periode wajib diisi"}
	}
	if kasbonSen < 0 {
		return 0, &ValidasiError{Pesan: "Potongan kasbon tidak boleh negatif"}
	}

	var id int64

	err := s.Repo.Transaksi(ctx, func(tx *repository.GajiTx) error {
		pekerja, err := tx.Pekerja(ctx, idPekerja)
		if err != nil {
			return err
		}

		periode, err := tx.Periode(ctx, idPeriode)
		if err != nil {
			return err
		}
		if periode.Status != "terbuka" {
			return &ValidasiError{Pesan: "Periode sudah ditutup"}
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

		hariAktif, err := hitungHariAktif(
			periode.TanggalAwal,
			periode.TanggalAkhir,
			pekerja.Masuk,
			keluar,
		)
		if err != nil {
			return err
		}
		if hariAktif == 0 && koreksiSen == 0 {
			return &ValidasiError{
				Pesan: "Pekerja tidak aktif pada periode ini dan tidak memiliki koreksi",
			}
		}

		total, rincian, err := hitungRincianDraf(
			ctx, tx, idPekerja, periode, pekerja,
		)
		if err != nil {
			return err
		}

		hasil, err := hitungTotalGaji(total, kasbonSen, koreksiSen)
		if err != nil {
			return err
		}

		gaji := model.Gaji{
			IDPekerja:           idPekerja,
			IDPeriode:           idPeriode,
			GajiPokok:           total.Pokok.FloatString(2),
			TotalLembur:         total.Lembur.FloatString(2),
			PotonganTelat:       total.PotonganTelat.FloatString(2),
			TotalPotonganKasbon: big.NewRat(kasbonSen, 100).FloatString(2),
			TotalKoreksi:        big.NewRat(koreksiSen, 100).FloatString(2),
			GajiBruto:           hasil.Bruto.FloatString(2),
			GajiBersih:          hasil.Bersih.FloatString(0),
			StatusBayar:         "belum",
		}

		id, err = simpanDraf(ctx, tx, gaji, rincian)
		return err
	})
	if err != nil {
		return 0, err
	}

	return id, nil
}

func hitungRincianDraf(
	ctx context.Context,
	tx *repository.GajiTx,
	idPekerja int,
	periode model.PeriodeGaji,
	pekerja repository.DataPekerja,
) (*rincianGaji, []model.DetailGaji, error) {
	const formatTanggal = "2006-01-02"

	daftar, err := tx.JadwalAbsensi(
		ctx, idPekerja, periode.TanggalAwal, periode.TanggalAkhir,
	)
	if err != nil {
		return nil, nil, err
	}

	masuk := pekerja.Masuk.Format(formatTanggal)
	keluar := ""
	if pekerja.Keluar.Valid {
		keluar = pekerja.Keluar.Time.Format(formatTanggal)
	}

	absensi := make(map[string]*model.Absensi)

	for _, jadwal := range daftar {
		tanggal := jadwal.Tanggal.Format(formatTanggal)

		if tanggal < masuk || (keluar != "" && tanggal > keluar) {
			return nil, nil, &ValidasiError{
				Pesan: "Ada jadwal di luar masa aktif pekerja",
			}
		}

		if jadwal.Absensi == nil || jadwal.Absensi.Status == "perlu_diperiksa" {
			return nil, nil, &ValidasiError{
				Pesan: fmt.Sprintf("Absensi tanggal %s belum lengkap", tanggal),
			}
		}

		// Validasi jam dan hitung menit menggunakan fungsi Absensi yang sudah ada.
		if err := hitungAbsensi(jadwal.Absensi, jadwal); err != nil {
			return nil, nil, err
		}

		absensi[tanggal] = jadwal.Absensi
	}

	total := &rincianGaji{}
	var rincian []model.DetailGaji

	akhir := periode.TanggalAkhir.Format(formatTanggal)
	hariBulan := int64(periode.TanggalAkhir.Day())

	pembagi := int64(8)
	if pekerja.Jenis == "bulanan" {
		pembagi = 173
	}

	for hari := periode.TanggalAwal; hari.Format(formatTanggal) <= akhir; hari = hari.AddDate(0, 0, 1) {
		tanggal := hari.Format(formatTanggal)

		if tanggal < masuk || (keluar != "" && tanggal > keluar) {
			continue
		}

		a, adaJadwal := absensi[tanggal]
		if pekerja.Jenis == "harian" && !adaJadwal {
			continue
		}

		upah, err := tx.UpahBerlaku(ctx, idPekerja, hari)
		if errors.Is(err, model.ErrNotFound) {
			return nil, nil, &ValidasiError{
				Pesan: fmt.Sprintf("Tarif upah tanggal %s belum tersedia", tanggal),
			}
		}
		if err != nil {
			return nil, nil, err
		}

		komponen, err := hitungKomponenHari(
			pekerja.Jenis, upah.NominalSen, hariBulan, a,
		)
		if err != nil {
			return nil, nil, err
		}

		detail := model.DetailGaji{
			IDUpah:        upah.ID,
			Tanggal:       hari,
			NominalUpah:   big.NewRat(upah.NominalSen, 100).FloatString(2),
			TarifPerJam:   big.NewRat(upah.NominalSen, 100*pembagi).FloatString(2),
			NilaiPokok:    komponen.Pokok.FloatString(2),
			JamLembur:     "0.000000",
			NilaiLembur:   komponen.Lembur.FloatString(2),
			PotonganTelat: komponen.PotonganTelat.FloatString(2),
		}

		if a != nil {
			detail.IDAbsensi = &a.ID
			detail.MenitTelat = a.MenitTerlambat
			detail.JamLembur = big.NewRat(int64(a.MenitLembur), 60).FloatString(6)
		}

		total.Pokok.Add(&total.Pokok, &komponen.Pokok)
		total.Lembur.Add(&total.Lembur, &komponen.Lembur)
		total.PotonganTelat.Add(&total.PotonganTelat, &komponen.PotonganTelat)

		rincian = append(rincian, detail)
	}

	return total, rincian, nil
}

func simpanDraf(
	ctx context.Context,
	tx *repository.GajiTx,
	g model.Gaji,
	rincian []model.DetailGaji,
) (int64, error) {
	if g.IDPekerja <= 0 || g.IDPeriode <= 0 {
		return 0, &ValidasiError{Pesan: "Pekerja dan periode wajib diisi"}
	}

	periode, err := tx.Periode(ctx, g.IDPeriode)
	if err != nil {
		return 0, err
	}
	if periode.Status != "terbuka" {
		return 0, &ValidasiError{Pesan: "Periode sudah ditutup"}
	}

	lama, err := tx.GajiPekerja(ctx, g.IDPekerja, g.IDPeriode)

	switch {
	case errors.Is(err, model.ErrNotFound):
		id, err := tx.TambahGaji(ctx, g)
		if err != nil {
			return 0, err
		}
		g.ID = int(id)

	case err != nil:
		return 0, err

	default:
		if lama.StatusBayar != "belum" {
			return 0, &ValidasiError{
				Pesan: "Gaji yang sudah dibayar tidak boleh dihitung ulang",
			}
		}

		g.ID = lama.ID

		if err := tx.UbahDraf(ctx, g); err != nil {
			return 0, err
		}
		if err := tx.HapusDetailDraf(ctx, g.ID); err != nil {
			return 0, err
		}
	}

	for _, detail := range rincian {
		detail.IDGaji = g.ID

		if err := tx.TambahDetail(ctx, detail); err != nil {
			return 0, err
		}
	}

	return int64(g.ID), nil
}
