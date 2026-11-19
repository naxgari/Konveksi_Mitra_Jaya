package service

import (
	"math/big"

	"github.com/anggyputra/go-class/internal/model"
)

type rincianGaji struct {
	Pokok         big.Rat
	Lembur        big.Rat
	PotonganTelat big.Rat
}

type hariGaji struct {
	UpahSen int64
	Absensi *model.Absensi
}

type totalGaji struct {
	Bruto  big.Rat
	Bersih big.Rat
}

// hitungGajiHarian menghitung gaji pokok, upah lembur, dan potongan
// keterlambatan berdasarkan upah harian serta data absensi.
// Perhitungan menggunakan big.Rat untuk menjaga presisi nominal.
func hitungGajiHarian(upahSen int64, absensi *model.Absensi) (*rincianGaji, error) {
	hasil := &rincianGaji{}

	if absensi == nil || (absensi.Status != "hadir" && absensi.Status != "tidak_hadir") {
		return nil, &ValidasiError{Pesan: "Absensi harus diperiksa sebelum gaji dihitung"}
	}
	if absensi.Status == "tidak_hadir" {
		return hasil, nil
	}
	if upahSen <= 0 {
		return nil, &ValidasiError{Pesan: "Upah harian yang berlaku belum tersedia"}
	}

	hasil.Pokok.SetFrac64(upahSen, 100)
	tarifPerJam := new(big.Rat).Quo(&hasil.Pokok, big.NewRat(8, 1))

	menitPertama := int64(min(absensi.MenitLembur, 60))
	menitBerikutnya := int64(max(absensi.MenitLembur-60, 0))

	// Tarif lembur: 1,5x pada jam pertama dan 2x pada jam berikutnya.
	jamLemburBerbobot := big.NewRat(3*menitPertama+4*menitBerikutnya, 120)
	hasil.Lembur.Mul(tarifPerJam, jamLemburBerbobot)

	if absensi.MenitTerlambat > 15 {
		hasil.PotonganTelat.Set(tarifPerJam)
	}

	return hasil, nil
}

// hitungGajiMingguan menjumlahkan komponen gaji harian
// menjadi total gaji pokok, lembur, dan potongan keterlambatan.
func hitungGajiMingguan(daftarHari []hariGaji) (*rincianGaji, error) {
	total := &rincianGaji{}

	for _, hari := range daftarHari {
		rincian, err := hitungGajiHarian(hari.UpahSen, hari.Absensi)
		if err != nil {
			return nil, err
		}

		total.Pokok.Add(&total.Pokok, &rincian.Pokok)
		total.Lembur.Add(&total.Lembur, &rincian.Lembur)
		total.PotonganTelat.Add(&total.PotonganTelat, &rincian.PotonganTelat)
	}

	return total, nil
}

func hitungPokokBulanan(upahSen, hariAktif, hariBulan int64) (*big.Rat, error) {
	if upahSen <= 0 {
		return nil, &ValidasiError{Pesan: "Upah bulanan belum tersedia"}
	}
	if hariBulan < 28 || hariBulan > 31 {
		return nil, &ValidasiError{Pesan: "Jumlah hari dalam bulan tidak valid"}
	}
	if hariAktif < 0 || hariAktif > hariBulan {
		return nil, &ValidasiError{Pesan: "Jumlah hari aktif tidak valid"}
	}

	upah := new(big.Rat).SetFrac64(upahSen, 100)
	proporsi := big.NewRat(hariAktif, hariBulan)

	return new(big.Rat).Mul(upah, proporsi), nil
}

func hitungTambahanBulanan(upahSen int64, absensi *model.Absensi) (*rincianGaji, error) {
	hasil := &rincianGaji{}

	if absensi == nil || (absensi.Status != "hadir" && absensi.Status != "tidak_hadir") {
		return nil, &ValidasiError{Pesan: "Absensi harus diperiksa sebelum gaji dihitung"}
	}
	if absensi.Status == "tidak_hadir" {
		return hasil, nil
	}
	if upahSen <= 0 {
		return nil, &ValidasiError{Pesan: "Upah bulanan belum tersedia"}
	}
	if absensi.MenitLembur < 0 || absensi.MenitTerlambat < 0 {
		return nil, &ValidasiError{Pesan: "Menit lembur dan terlambat tidak boleh negatif"}
	}

	upah := new(big.Rat).SetFrac64(upahSen, 100)
	tarifPerJam := new(big.Rat).Quo(upah, big.NewRat(173, 1))

	menitPertama := int64(min(absensi.MenitLembur, 60))
	menitBerikutnya := int64(max(absensi.MenitLembur-60, 0))
	jamBerbobot := big.NewRat(3*menitPertama+4*menitBerikutnya, 120)

	hasil.Lembur.Mul(tarifPerJam, jamBerbobot)

	if absensi.MenitTerlambat > 15 {
		hasil.PotonganTelat.Set(tarifPerJam)
	}

	return hasil, nil
}

func hitungGajiBulanan(daftarHari []hariGaji, hariBulan int64) (*rincianGaji, error) {
	if hariBulan < 28 || hariBulan > 31 {
		return nil, &ValidasiError{Pesan: "Jumlah hari dalam bulan tidak valid"}
	}
	if int64(len(daftarHari)) > hariBulan {
		return nil, &ValidasiError{Pesan: "Jumlah hari aktif melebihi jumlah hari dalam bulan"}
	}

	total := &rincianGaji{}

	for _, hari := range daftarHari {
		pokok, err := hitungPokokBulanan(hari.UpahSen, 1, hariBulan)
		if err != nil {
			return nil, err
		}
		total.Pokok.Add(&total.Pokok, pokok)

		if hari.Absensi != nil {
			tambahan, err := hitungTambahanBulanan(hari.UpahSen, hari.Absensi)
			if err != nil {
				return nil, err
			}

			total.Lembur.Add(&total.Lembur, &tambahan.Lembur)
			total.PotonganTelat.Add(&total.PotonganTelat, &tambahan.PotonganTelat)
		}
	}

	return total, nil
}

func hitungTotalGaji(rincian *rincianGaji, kasbonSen, koreksiSen int64) (*totalGaji, error) {
	if rincian == nil {
		return nil, &ValidasiError{Pesan: "Rincian gaji belum tersedia"}
	}
	if kasbonSen < 0 {
		return nil, &ValidasiError{Pesan: "Potongan kasbon tidak boleh negatif"}
	}

	hasil := &totalGaji{}
	kasbon := big.NewRat(kasbonSen, 100)
	koreksi := big.NewRat(koreksiSen, 100)

	hasil.Bruto.Add(&rincian.Pokok, &rincian.Lembur)

	hasil.Bersih.Sub(&hasil.Bruto, &rincian.PotonganTelat)
	hasil.Bersih.Sub(&hasil.Bersih, kasbon)
	hasil.Bersih.Add(&hasil.Bersih, koreksi)

	return hasil, nil
}

func hitungKomponenHari(
	jenis string,
	upahSen, hariBulan int64,
	absensi *model.Absensi,
) (*rincianGaji, error) {
	if jenis == "harian" {
		return hitungGajiHarian(upahSen, absensi)
	}

	pokok, err := hitungPokokBulanan(upahSen, 1, hariBulan)
	if err != nil {
		return nil, err
	}

	hasil := &rincianGaji{}

	if absensi != nil {
		hasil, err = hitungTambahanBulanan(upahSen, absensi)
		if err != nil {
			return nil, err
		}
	}

	hasil.Pokok.Set(pokok)

	return hasil, nil
}
