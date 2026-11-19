package service

import (
	"context"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/anggyputra/go-class/internal/model"
	"github.com/anggyputra/go-class/internal/repository"
)

// KasbonService menangani pengajuan kasbon, pelacakan saldo, dan pemotongan
// kasbon otomatis saat hitung gaji (Aturan Bisnis AB 08).
type KasbonService struct{ Repo *repository.KasbonRepo }

// BatasPotonganPersen adalah proporsi maksimum gaji bruto yang boleh dipotong
// untuk kasbon (AB 08): 30%.
const BatasPotonganPersen = 0.30
const batasPotonganBasisPoin = int64(BatasPotonganPersen * 10000)

func uangDalamSen(x float64) int64 {
	return int64(math.Round(x * 100))
}

func nominalDuaDesimalValid(x float64) bool {
	return !math.IsNaN(x) && !math.IsInf(x, 0) && math.Abs(x-float64(uangDalamSen(x))/100) < 1e-8
}

// RencanaPotong adalah satu baris rencana pemotongan kasbon (hasil perhitungan murni).
type RencanaPotong struct {
	IDKasbon int
	Jumlah   float64
}

// hitungRencanaPotong menghitung potongan FIFO dengan batas 30% bruto dibulatkan
// ke bawah ke rupiah, serta tidak melebihi saldo atau gaji bersih yang tersedia.
// Fungsi ini murni (tanpa akses DB) sehingga mudah diuji.
func hitungRencanaPotong(bruto, tersedia float64, kandidat []repository.KasbonKandidat) []RencanaPotong {
	batasSen := (uangDalamSen(bruto) * batasPotonganBasisPoin / 10000 / 100) * 100
	sisaSen := uangDalamSen(tersedia)
	if sisaSen < 0 {
		sisaSen = 0
	}
	if batasSen < sisaSen {
		sisaSen = batasSen
	}
	out := make([]RencanaPotong, 0, len(kandidat))
	for _, k := range kandidat {
		if sisaSen <= 0 {
			break
		}
		saldoSen := uangDalamSen(k.Saldo)
		if saldoSen <= 0 {
			continue
		}
		ambilSen := saldoSen
		if ambilSen > sisaSen {
			ambilSen = sisaSen
		}
		out = append(out, RencanaPotong{IDKasbon: k.ID, Jumlah: float64(ambilSen) / 100})
		sisaSen -= ambilSen
	}
	return out
}

// KasbonDaftar mengembalikan daftar kasbon beserta saldo.
func (s *KasbonService) KasbonDaftar(ctx context.Context) ([]model.Kasbon, error) {
	return s.Repo.KasbonDaftar(ctx)
}

// KasbonDetail mengembalikan satu kasbon beserta riwayat pemotongannya.
func (s *KasbonService) KasbonDetail(ctx context.Context, id int) (model.Kasbon, []model.PotonganKasbon, error) {
	if id <= 0 {
		return model.Kasbon{}, nil, &ValidasiError{"ID kasbon tidak valid"}
	}
	k, err := s.Repo.KasbonDetail(ctx, id)
	if err != nil {
		return model.Kasbon{}, nil, err
	}
	p, err := s.Repo.PotonganByKasbon(ctx, id)
	return k, p, err
}

// PekerjaPilihan mengembalikan daftar pekerja aktif untuk dropdown form kasbon.
func (s *KasbonService) PekerjaPilihan(ctx context.Context) ([]model.PekerjaJadwal, error) {
	return s.Repo.PekerjaAktif(ctx)
}

// KasbonBuat memvalidasi & mencatat pengajuan kasbon baru.
func (s *KasbonService) KasbonBuat(ctx context.Context, k model.Kasbon) (int64, error) {
	if err := validasiKasbon(k); err != nil {
		return 0, err
	}
	ada, err := s.Repo.PekerjaAdaAktif(ctx, k.IDPekerja)
	if err != nil {
		return 0, err
	}
	if !ada {
		return 0, &ValidasiError{"Pekerja tidak ditemukan atau sudah berhenti"}
	}
	id, err := s.Repo.KasbonBuat(ctx, k)
	return id, errorAturan(err)
}

func validasiKasbon(k model.Kasbon) error {
	if k.IDPekerja <= 0 {
		return &ValidasiError{"Pekerja wajib dipilih"}
	}
	if k.Tanggal.IsZero() {
		return &ValidasiError{"Tanggal pengambilan wajib diisi"}
	}
	if k.Jumlah <= 0 || !nominalDuaDesimalValid(k.Jumlah) {
		return &ValidasiError{"Jumlah kasbon harus lebih dari 0"}
	}
	if utf8.RuneCountInString(k.Keterangan) > 255 {
		return &ValidasiError{"Keterangan maksimal 255 karakter"}
	}
	return nil
}

// KasbonHapus menghapus kasbon yang belum pernah dipotong. Kasbon yang sudah
// dipotong tidak boleh dihapus (FK RESTRICT) agar riwayat gaji tetap utuh.
func (s *KasbonService) KasbonHapus(ctx context.Context, id int) error {
	if id <= 0 {
		return &ValidasiError{"ID kasbon tidak valid"}
	}
	dipotong, err := s.Repo.JumlahPotonganKasbon(ctx, id)
	if err != nil {
		return err
	}
	if dipotong > 0 {
		return &ValidasiError{"Kasbon yang sudah dipotong tidak dapat dihapus"}
	}
	return errorAturan(s.Repo.KasbonHapus(ctx, id))
}

// PotongOtomatis menjalankan pemotongan kasbon otomatis pada suatu gaji (AB 08):
//   - batas maksimal 30% gaji bruto, dibulatkan ke bawah ke rupiah;
//   - memotong kasbon tertua lebih dulu (FIFO) yang tanggalnya tidak melewati akhir periode;
//   - idempoten: menghapus potongan lama pada gaji lalu menghitung ulang.
//
// Hanya boleh dijalankan saat gaji berstatus "belum" (draf, sebelum dibayar).
// Kontrak gaji_bersih: gaji_bruto - potongan_telat - total_potongan_kasbon + total_koreksi.
// Dipanggil oleh engine hitung gaji (modul Daffa) setelah gaji_bruto terhitung.
func (s *KasbonService) PotongOtomatis(ctx context.Context, idGaji int) ([]RencanaPotong, error) {
	if idGaji <= 0 {
		return nil, &ValidasiError{"ID gaji tidak valid"}
	}
	var rencana []RencanaPotong
	err := s.Repo.Transaksi(ctx, func(tx *repository.KasbonTx) error {
		d, e := tx.Gaji(ctx, idGaji)
		if e != nil {
			return e
		}
		if d.StatusPeriode != "terbuka" {
			return &ValidasiError{"Periode gaji sudah ditutup"}
		}
		if d.StatusBayar != "belum" {
			return &ValidasiError{"Pemotongan kasbon hanya untuk gaji yang belum dibayar"}
		}
		koreksi, e := tx.KoreksiDisetujuiUntuk(ctx, d.IDPekerja, idGaji)
		if e != nil {
			return e
		}
		tersediaSen := uangDalamSen(d.GajiBruto) - uangDalamSen(d.PotonganTelat) + uangDalamSen(d.TotalKoreksi)
		for _, k := range koreksi {
			tersediaSen += uangDalamSen(k.NilaiSelisih)
		}
		if tersediaSen < 0 {
			return &ValidasiError{"Gaji sebelum potongan kasbon bernilai negatif; perlu diperiksa"}
		}
		if e = tx.HapusPotonganByGaji(ctx, idGaji); e != nil {
			return e
		}
		if d.GajiBruto > 0 && tersediaSen > 0 {
			kandidat, err := tx.KasbonUntukPotong(ctx, d.IDPekerja, d.TanggalAkhir)
			if err != nil {
				return err
			}
			rencana = hitungRencanaPotong(d.GajiBruto, float64(tersediaSen)/100, kandidat)
			for _, r := range rencana {
				if err = tx.TambahPotongan(ctx, idGaji, r.IDKasbon, r.Jumlah); err != nil {
					return err
				}
			}
		}
		for _, k := range koreksi {
			if e = tx.TautkanKoreksi(ctx, k.ID, idGaji); e != nil {
				return e
			}
		}
		totalPotongan, e := tx.TotalPotonganByGaji(ctx, idGaji)
		if e != nil {
			return e
		}
		totalKoreksi, e := tx.TotalKoreksiByGaji(ctx, idGaji)
		if e != nil {
			return e
		}
		return tx.UpdateGajiPotongan(ctx, idGaji, d, totalPotongan, totalKoreksi)
	})
	return rencana, errorAturan(err)
}

func (s *KasbonService) KoreksiDaftar(ctx context.Context) ([]model.KoreksiGaji, error) {
	return s.Repo.KoreksiDaftar(ctx)
}

func (s *KasbonService) GajiDibayarPilihan(ctx context.Context) ([]model.PilihanGaji, error) {
	return s.Repo.PilihanGajiDibayar(ctx)
}

func (s *KasbonService) KoreksiBuat(ctx context.Context, k model.KoreksiGaji) (int64, error) {
	k.Alasan = strings.TrimSpace(k.Alasan)
	if k.IDGajiAsal <= 0 || k.Tanggal.IsZero() || k.NilaiSelisih == 0 || !nominalDuaDesimalValid(k.NilaiSelisih) || k.Alasan == "" || utf8.RuneCountInString(k.Alasan) > 255 {
		return 0, &ValidasiError{"Gaji asal, tanggal, nilai selisih, dan alasan wajib valid"}
	}
	dibayar, err := s.Repo.GajiAsalDibayar(ctx, k.IDGajiAsal)
	if err != nil {
		return 0, err
	}
	if !dibayar {
		return 0, &ValidasiError{"Gaji asal harus berstatus sudah dibayar"}
	}
	id, err := s.Repo.KoreksiBuat(ctx, k)
	return id, errorAturan(err)
}

func (s *KasbonService) KoreksiSetujui(ctx context.Context, id, idPemilik int) error {
	if id <= 0 || idPemilik <= 0 {
		return &ValidasiError{"ID koreksi atau pemilik tidak valid"}
	}
	return errorAturan(s.Repo.KoreksiSetujui(ctx, id, idPemilik))
}

func (s *KasbonService) KoreksiHapus(ctx context.Context, id int) error {
	if id <= 0 {
		return &ValidasiError{"ID koreksi tidak valid"}
	}
	return errorAturan(s.Repo.KoreksiHapus(ctx, id))
}

func (s *KasbonService) Slip(ctx context.Context, idGaji int) (model.SlipGaji, error) {
	if idGaji <= 0 {
		return model.SlipGaji{}, &ValidasiError{"ID gaji tidak valid"}
	}
	return s.Repo.AmbilSlip(ctx, idGaji)
}

func (s *KasbonService) PeriodePilihan(ctx context.Context) ([]model.PilihanPeriode, error) {
	return s.Repo.PilihanPeriode(ctx)
}

func (s *KasbonService) Rekap(ctx context.Context, idPeriode int, jenisPekerja string) ([]model.RekapBaris, model.RingkasanRekap, error) {
	if idPeriode < 0 {
		return nil, model.RingkasanRekap{}, &ValidasiError{"Periode tidak valid"}
	}
	if jenisPekerja != "" && jenisPekerja != "harian" && jenisPekerja != "bulanan" {
		return nil, model.RingkasanRekap{}, &ValidasiError{"Jenis pekerja tidak valid"}
	}
	return s.Repo.RekapDaftar(ctx, idPeriode, jenisPekerja)
}
