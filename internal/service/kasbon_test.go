package service

import (
	"math"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/anggyputra/go-class/internal/model"
	"github.com/anggyputra/go-class/internal/repository"
)

func TestHitungRencanaPotongBatas30PersenDanFIFO(t *testing.T) {
	kandidat := []repository.KasbonKandidat{
		{ID: 1, Saldo: 200_000},
		{ID: 2, Saldo: 200_000},
		{ID: 3, Saldo: 0},
	}

	got := hitungRencanaPotong(1_000_000, 1_000_000, kandidat)
	want := []RencanaPotong{{IDKasbon: 1, Jumlah: 200_000}, {IDKasbon: 2, Jumlah: 100_000}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("hitungRencanaPotong() = %#v, want %#v", got, want)
	}
}

func TestHitungRencanaPotongBulatKeRupiahDanBatasiGajiTersedia(t *testing.T) {
	kandidat := []repository.KasbonKandidat{{ID: 1, Saldo: 1_000}}

	got := hitungRencanaPotong(1_234.57, 1_000, kandidat)
	want := []RencanaPotong{{IDKasbon: 1, Jumlah: 370}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("batas 30%% = %#v, want %#v", got, want)
	}

	got = hitungRencanaPotong(1_000, 120.45, kandidat)
	want = []RencanaPotong{{IDKasbon: 1, Jumlah: 120.45}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("batas gaji tersedia = %#v, want %#v", got, want)
	}
}

func TestHitungRencanaPotongMenolakNilaiSebelumPotonganNegatif(t *testing.T) {
	got := hitungRencanaPotong(1_000_000, -1, []repository.KasbonKandidat{{ID: 1, Saldo: 500_000}})
	if len(got) != 0 {
		t.Fatalf("hitungRencanaPotong() = %#v, want no deductions", got)
	}
}

func TestValidasiKasbonMenolakNominalTidakValid(t *testing.T) {
	k := model.Kasbon{IDPekerja: 1, Tanggal: time.Now(), Jumlah: 1}
	for _, jumlah := range []float64{0, -1, 0.001, math.NaN(), math.Inf(1), 1e14} {
		k.Jumlah = jumlah
		if err := validasiKasbon(k); err == nil {
			t.Errorf("validasiKasbon(Jumlah=%v) tidak menolak nominal", jumlah)
		}
	}
}

func TestHitungAlokasiKasbonIntegrasi(t *testing.T) {
	// Contoh Ayu di dokumen tahap 1: bruto 772.500, telat 15.000, saldo 300.000 -> batas 30% = 231.750.
	bruto := big.NewRat(772500, 1)
	telat := big.NewRat(15000, 1)
	alokasi, totalSen, negatif := hitungAlokasiKasbon(bruto, telat, 50000_00, []repository.KasbonSaldo{
		{ID: 1, SaldoSen: 200000_00},
		{ID: 2, SaldoSen: 100000_00},
	})
	if negatif {
		t.Fatal("seharusnya tidak negatif")
	}
	if totalSen != 231750_00 {
		t.Fatalf("totalSen = %d, want 23175000", totalSen)
	}
	if len(alokasi) != 2 || alokasi[0].JumlahSen != 200000_00 || alokasi[1].JumlahSen != 31750_00 {
		t.Fatalf("alokasi FIFO = %+v", alokasi)
	}

	// Nilai sebelum kasbon negatif (koreksi negatif besar) harus ditandai sebelumNegatif = true.
	alokasiNeg, totalNeg, isNeg := hitungAlokasiKasbon(big.NewRat(100000, 1), big.NewRat(0, 1), -150000_00, []repository.KasbonSaldo{
		{ID: 1, SaldoSen: 50000_00},
	})
	if !isNeg || totalNeg != 0 || len(alokasiNeg) != 0 {
		t.Fatalf("nilai sebelum kasbon negatif seharusnya menahan kasbon: neg=%v total=%d alokasi=%+v", isNeg, totalNeg, alokasiNeg)
	}
}
