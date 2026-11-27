package service

import (
	"testing"
	"math/big"
	"github.com/anggyputra/go-class/internal/model"
	"time"
	"context"
)

func TestHitungGajiHarian(t *testing.T) {
	absensi := &model.Absensi{
		Status:         "hadir",
		MenitTerlambat: 20,
		MenitLembur:    120,
	}

	hasil, err := hitungGajiHarian(12_000_000, absensi)
	if err != nil {
		t.Fatal(err)
	}

	if hasil.Pokok.RatString() != "120000" ||
		hasil.Lembur.RatString() != "52500" ||
		hasil.PotonganTelat.RatString() != "15000" {
		t.Fatalf("pokok = %s, lembur = %s, potongan telat = %s",
			hasil.Pokok.RatString(),
			hasil.Lembur.RatString(),
			hasil.PotonganTelat.RatString(),
		)
	}
}
func TestBatasPotonganTelat(t *testing.T) {
	kasus := []struct {
		menit int
		mau   string
	}{
		{15, "0"},
		{16, "15000"},
	}

	for _, k := range kasus {
		hasil, err := hitungGajiHarian(12_000_000, &model.Absensi{
			Status:          "hadir",
			MenitTerlambat: k.menit,
		})
		if err != nil {
			t.Fatal(err)
		}
		if dapat := hasil.PotonganTelat.RatString(); dapat != k.mau {
			t.Errorf("telat %d menit: potongan Rp%s, mau Rp%s", k.menit, dapat, k.mau)
		}
	}
}


func TestHitungGajiMingguan(t *testing.T) {
	hari := make([]hariGaji, 6)
	for i := range hari {
		hari[i] = hariGaji{
			UpahSen: 12_000_000,
			Absensi: &model.Absensi{Status: "hadir"},
		}
	}
	hari[0].Absensi.MenitTerlambat = 20
	hari[0].Absensi.MenitLembur = 120

	total, err := hitungGajiMingguan(hari)
	if err != nil {
		t.Fatal(err)
	}

	bruto := new(big.Rat).Add(&total.Pokok, &total.Lembur)
	setelahPotonganTelat := new(big.Rat).Sub(bruto, &total.PotonganTelat)

	if bruto.RatString() != "772500" || setelahPotonganTelat.RatString() != "757500" {
		t.Fatalf("bruto = %s, setelah potongan telat = %s",
			bruto.RatString(), setelahPotonganTelat.RatString())
	}
}


func TestPokokBulanan(t *testing.T) {
	kasus := []struct {
		nama      string
		hariAktif int64
		mau       string
	}{
		{"Bima masuk tengah bulan", 16, "1600000"},
		{"aktif sebulan penuh", 31, "3100000"},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			hasil, err := hitungPokokBulanan(310_000_000, k.hariAktif, 31)
			if err != nil {
				t.Fatal(err)
			}
			if hasil.RatString() != k.mau {
				t.Fatalf("pokok = %s, mau %s", hasil.RatString(), k.mau)
			}
		})
	}
}

func TestHitungTambahanBulanan(t *testing.T) {
	kasus := []struct {
		nama         string
		menitLembur  int
		menitTelat   int
		mauLembur    string
		mauPotongan  string
	}{
		{"lembur dua jam dan telat", 120, 20, "70000", "20000"},
		{"lembur setengah jam", 30, 0, "15000", "0"},
		{"telat tepat 15 menit", 0, 15, "0", "0"},
		{"telat 16 menit", 0, 16, "0", "20000"},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			hasil, err := hitungTambahanBulanan(346_000_000, &model.Absensi{
				Status:         "hadir",
				MenitLembur:    k.menitLembur,
				MenitTerlambat: k.menitTelat,
			})
			if err != nil {
				t.Fatal(err)
			}

			if hasil.Lembur.RatString() != k.mauLembur {
				t.Errorf("lembur = %s, mau %s",
					hasil.Lembur.RatString(), k.mauLembur)
			}
			if hasil.PotonganTelat.RatString() != k.mauPotongan {
				t.Errorf("potongan telat = %s, mau %s",
					hasil.PotonganTelat.RatString(), k.mauPotongan)
			}
		})
	}
}


func TestHitungGajiBulanan(t *testing.T) {
	hari := make([]hariGaji, 30)
	for i := range hari {
		hari[i].UpahSen = 346_000_000
	}

	hari[0].Absensi = &model.Absensi{
		Status:         "hadir",
		MenitLembur:    120,
		MenitTerlambat: 20,
	}

	total, err := hitungGajiBulanan(hari, 30)
	if err != nil {
		t.Fatal(err)
	}

	if total.Pokok.RatString() != "3460000" ||
		total.Lembur.RatString() != "70000" ||
		total.PotonganTelat.RatString() != "20000" {
		t.Fatalf("pokok = %s, lembur = %s, potongan telat = %s",
			total.Pokok.RatString(),
			total.Lembur.RatString(),
			total.PotonganTelat.RatString())
	}
}


func TestHitungTotalGaji(t *testing.T) {
	rincian := &rincianGaji{}
	rincian.Pokok.SetInt64(720000)
	rincian.Lembur.SetInt64(52500)
	rincian.PotonganTelat.SetInt64(15000)

	kasus := []struct {
		nama       string
		kasbonSen  int64
		koreksiSen int64
		mauBersih  string
	}{
		{"tanpa kasbon dan koreksi", 0, 0, "757500"},
		{"dengan kasbon", 23_175_000, 0, "525750"},
		{"koreksi menambah gaji", 23_175_000, 5_000_000, "575750"},
		{"koreksi mengurangi gaji", 23_175_000, -5_000_000, "475750"},
		{"pecahan di bawah setengah", 0, 49, "757500"},
		{"pecahan tepat setengah", 0, 50, "757501"},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			hasil, err := hitungTotalGaji(rincian, k.kasbonSen, k.koreksiSen)
			if err != nil {
				t.Fatal(err)
			}

			if hasil.Bruto.RatString() != "772500" {
				t.Errorf("bruto = %s, mau 772500", hasil.Bruto.RatString())
			}

			if dapat := hasil.Bersih.FloatString(0); dapat != k.mauBersih {
				t.Errorf("bersih = %s, mau %s", dapat, k.mauBersih)
			}
		})
	}
}

func TestValidasiPeriodeMingguan(t *testing.T) {
	senin := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	sabtu := senin.AddDate(0, 0, 5)

	kasus := []struct {
		nama                string
		mulai, akhir, bayar time.Time
		mauError            bool
	}{
		{"periode sesuai", senin, sabtu, sabtu, false},
		{"mulai Selasa", senin.AddDate(0, 0, 1), sabtu, sabtu, true},
		{"akhir Minggu", senin, sabtu.AddDate(0, 0, 1), sabtu, true},
		{"bayar Jumat", senin, sabtu, sabtu.AddDate(0, 0, -1), true},
		{"tanggal mulai kosong", time.Time{}, sabtu, sabtu, true},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			err := validasiPeriodeMingguan(k.mulai, k.akhir, k.bayar)
			if (err != nil) != k.mauError {
				t.Fatalf("error = %v, mau error = %t", err, k.mauError)
			}
		})
	}
}

func TestValidasiPeriodeBulanan(t *testing.T) {
	november := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	akhirNovember := time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC)

	februari := time.Date(2028, 2, 1, 0, 0, 0, 0, time.UTC)
	akhirFebruari := time.Date(2028, 2, 29, 0, 0, 0, 0, time.UTC)

	kasus := []struct {
		nama                string
		mulai, akhir, bayar time.Time
		mauError            bool
	}{
		{"November lengkap", november, akhirNovember, akhirNovember, false},
		{"Februari kabisat", februari, akhirFebruari, akhirFebruari, false},
		{"mulai tanggal 2", november.AddDate(0, 0, 1), akhirNovember, akhirNovember, true},
		{"akhir terlalu awal", november, akhirNovember.AddDate(0, 0, -1), akhirNovember, true},
		{"pembayaran kosong", november, akhirNovember, time.Time{}, true},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			err := validasiPeriodeBulanan(k.mulai, k.akhir, k.bayar)
			if (err != nil) != k.mauError {
				t.Fatalf("error = %v, mau error = %t", err, k.mauError)
			}
		})
	}
}

func TestHitungHariAktif(t *testing.T) {
	tanggal := func(bulan time.Month, hari int) time.Time {
		return time.Date(2026, bulan, hari, 0, 0, 0, 0, time.UTC)
	}

	kasus := []struct {
		nama          string
		masuk, keluar time.Time
		ingin         int64
	}{
		{"sebulan penuh", tanggal(time.October, 1), time.Time{}, 30},
		{"masuk tengah bulan", tanggal(time.November, 16), time.Time{}, 15},
		{"keluar tengah bulan", tanggal(time.October, 1), tanggal(time.November, 15), 15},
		{"masuk dan keluar hari sama", tanggal(time.November, 16), tanggal(time.November, 16), 1},
		{"belum masuk", tanggal(time.December, 1), time.Time{}, 0},
		{"sudah keluar", tanggal(time.October, 1), tanggal(time.October, 31), 0},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			hasil, err := hitungHariAktif(
				tanggal(time.November, 1),
				tanggal(time.November, 30),
				k.masuk,
				k.keluar,
			)
			if err != nil {
				t.Fatal(err)
			}
			if hasil != k.ingin {
				t.Fatalf("hari aktif = %d, ingin %d", hasil, k.ingin)
			}
		})
	}
}

func TestValidasiPeriode(t *testing.T) {
	tanggal := func(hari int) time.Time {
		return time.Date(2026, time.November, hari, 0, 0, 0, 0, time.Local)
	}

	baru := model.PeriodeGaji{
		Jenis:               "mingguan",
		TanggalAwal:         tanggal(16),
		TanggalAkhir:        tanggal(21),
		TanggalBayarRencana: tanggal(21),
	}

	kasus := []struct {
		nama      string
		daftar    []model.PeriodeGaji
		inginErr  bool
	}{
		{
			nama: "belum ada periode",
		},
		{
			nama:     "periode sama",
			daftar:   []model.PeriodeGaji{baru},
			inginErr: true,
		},
		{
			nama: "minggu sebelumnya",
			daftar: []model.PeriodeGaji{{
				Jenis:        "mingguan",
				TanggalAwal:  tanggal(9),
				TanggalAkhir: tanggal(14),
			}},
		},
		{
			nama: "jenis berbeda boleh beririsan",
			daftar: []model.PeriodeGaji{{
				Jenis:        "bulanan",
				TanggalAwal:  tanggal(1),
				TanggalAkhir: tanggal(30),
			}},
		},
		{
			nama: "periode tertutup tetap ditolak",
			daftar: []model.PeriodeGaji{{
				Jenis:        "mingguan",
				TanggalAwal:  tanggal(16),
				TanggalAkhir: tanggal(21),
				Status:       "tertutup",
			}},
			inginErr: true,
		},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			err := validasiPeriode(baru, k.daftar)

			if (err != nil) != k.inginErr {
				t.Fatalf("error = %v, inginErr = %v", err, k.inginErr)
			}
		})
	}
}

func TestValidasiPembayaran(t *testing.T) {
	kasus := []struct {
		nama          string
		statusPeriode string
		statusBayar   string
		inginErr      bool
	}{
		{"boleh dibayar", "terbuka", "belum", false},
		{"sudah dibayar", "terbuka", "dibayar", true},
		{"periode tertutup", "tertutup", "belum", true},
		{"status periode kosong", "", "belum", true},
		{"status bayar kosong", "terbuka", "", true},
		{"status bayar tidak dikenal", "terbuka", "lunas", true},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			err := validasiPembayaran(k.statusPeriode, k.statusBayar)

			if (err != nil) != k.inginErr {
				t.Fatalf("error = %v, inginErr = %v", err, k.inginErr)
			}
		})
	}
}

func TestValidasiPenutupanPeriode(t *testing.T) {
	kasus := []struct {
		nama         string
		peran        model.PeranPengguna
		status       string
		statusGaji   []string
		inginErr     bool
	}{
		{
			nama:       "pemilik dan semua gaji dibayar",
			peran:      model.PeranPemilik,
			status:     "terbuka",
			statusGaji: []string{"dibayar", "dibayar"},
		},
		{
			nama:       "masih ada gaji belum dibayar",
			peran:      model.PeranPemilik,
			status:     "terbuka",
			statusGaji: []string{"dibayar", "belum"},
			inginErr:   true,
		},
		{
			nama:       "admin tidak boleh menutup",
			peran:      model.PeranAdmin,
			status:     "terbuka",
			statusGaji: []string{"dibayar"},
			inginErr:   true,
		},
		{
			nama:       "periode sudah tertutup",
			peran:      model.PeranPemilik,
			status:     "tertutup",
			statusGaji: []string{"dibayar"},
			inginErr:   true,
		},
		{
			nama:       "periode kosong tanpa gaji",
			peran:      model.PeranPemilik,
			status:     "terbuka",
			statusGaji: nil,
			inginErr:   true,
		},
		{
			nama:       "status gaji tidak valid",
			peran:      model.PeranPemilik,
			status:     "terbuka",
			statusGaji: []string{""},
			inginErr:   true,
		},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			err := validasiPenutupanPeriode(k.peran, k.status, k.statusGaji)

			if (err != nil) != k.inginErr {
				t.Fatalf("error = %v, inginErr = %v", err, k.inginErr)
			}
		})
	}
}

func TestDetailPeriodeIDTidakValid(t *testing.T) {
	s := &PeriodeGajiService{}

	for _, id := range []int{0, -1} {
		_, err := s.Detail(context.Background(), id)
		if err == nil {
			t.Errorf("ID %d seharusnya ditolak", id)
		}
	}
}

func TestBuatPeriodeTidakValid(t *testing.T) {
	s := &PeriodeGajiService{}

	id, err := s.Buat(context.Background(), model.PeriodeGaji{
		Jenis: "tahunan",
	})

	if err == nil {
		t.Fatal("jenis periode tidak valid seharusnya ditolak")
	}
	if id != 0 {
		t.Fatalf("ID = %d, seharusnya 0", id)
	}
}