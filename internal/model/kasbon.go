package model

import "time"

// Kasbon merepresentasikan pengajuan/pengambilan pinjaman pekerja (tabel KASBON).
// Saldo = Jumlah - SudahDipotong; potongan mengurangi saldo setelah gaji dibayar.
type Kasbon struct {
	ID            int
	IDPekerja     int
	NamaPekerja   string
	JenisPekerja  string
	Tanggal       time.Time
	Jumlah        float64
	Keterangan    string
	SudahDipotong float64
	Saldo         float64
}

// PotonganKasbon merepresentasikan pemotongan kasbon pada suatu gaji (tabel POTONGAN_KASBON).
// Dipakai pada halaman detail kasbon untuk menampilkan riwayat pemotongan.
type PotonganKasbon struct {
	ID           int
	IDGaji       int
	IDKasbon     int
	Jumlah       float64
	PeriodeLabel string // contoh: "05 Feb 2091 · mingguan"
	StatusBayar  string // belum | dibayar
}

// KoreksiGaji merepresentasikan koreksi antarperiode (tabel KOREKSI_GAJI).
// IDGajiTujuan == 0 berarti belum ditautkan ke gaji tujuan (status diajukan/disetujui).
type KoreksiGaji struct {
	ID            int
	IDGajiAsal    int
	IDGajiTujuan  int
	NamaPekerja   string
	PeriodeAsal   string // label tanggal awal periode asal
	PeriodeTujuan string // label tanggal awal periode tujuan, atau "—" bila belum ditautkan
	DicatatOleh   int
	NamaPencatat  string
	DisetujuiOleh int
	NamaPenyetuju string
	Tanggal       time.Time
	NilaiSelisih  float64 // positif menambah gaji tujuan, negatif mengurangi
	Alasan        string
	Status        string // diajukan | disetujui | diterapkan
	DisetujuiPada time.Time
}

// PilihanGaji untuk dropdown gaji asal (hanya gaji berstatus dibayar) pada form koreksi.
type PilihanGaji struct {
	ID    int
	Label string
}

// PilihanPeriode untuk filter laporan rekap.
type PilihanPeriode struct {
	ID    int
	Label string
}

// SlipGaji adalah agregat data untuk slip gaji individual (siap cetak).
type SlipGaji struct {
	Gaji           SlipGajiRingkas
	Pekerja        SlipPekerja
	Periode        SlipPeriode
	Detail         []SlipBaris
	PotonganKasbon []SlipPotongan
	Koreksi        []SlipKoreksi
	BisaPotong     bool // true bila status_bayar=belum (tombol potong/terapkan tampil)
}

type SlipGajiRingkas struct {
	ID                  int
	GajiPokok           float64
	TotalLembur         float64
	PotonganTelat       float64
	TotalPotonganKasbon float64
	TotalKoreksi        float64
	GajiBruto           float64
	GajiBersih          float64
	StatusBayar         string
	DibayarPada         time.Time
}

type SlipPekerja struct {
	ID      int
	Nama    string
	Jenis   string
	Jabatan string
}

type SlipPeriode struct {
	ID           int
	JenisPeriode string
	TanggalAwal  time.Time
	TanggalAkhir time.Time
	Status       string
}

// SlipBaris adalah satu baris rincian gaji per tanggal (dari DETAIL_GAJI).
type SlipBaris struct {
	Tanggal       time.Time
	NamaShift     string // kosong bila tanpa absensi (pekerja bulanan)
	JamMasuk      string
	JamKeluar     string
	NilaiPokok    float64
	MenitTelat    int
	JamLembur     float64
	NilaiLembur   float64
	PotonganTelat float64
}

// SlipPotongan adalah baris potongan kasbon pada slip.
type SlipPotongan struct {
	TanggalKasbon time.Time
	Jumlah        float64
	Keterangan    string
}

// SlipKoreksi adalah baris koreksi yang diterapkan pada slip.
type SlipKoreksi struct {
	Tanggal      time.Time
	NilaiSelisih float64
	Alasan       string
	Status       string
}

// RekapBaris adalah satu baris laporan rekap biaya gaji & lembur per periode.
type RekapBaris struct {
	IDGaji         int
	IDPekerja      int
	NamaPekerja    string
	Periode        string
	JenisPekerja   string
	GajiPokok      float64
	TotalLembur    float64
	PotonganTelat  float64
	PotonganKasbon float64
	TotalKoreksi   float64
	GajiBruto      float64
	GajiBersih     float64
	StatusBayar    string
}

// RingkasanRekap adalah totalan laporan rekap.
type RingkasanRekap struct {
	TotalGajiPokok      float64
	TotalLembur         float64
	TotalPotonganTelat  float64
	TotalPotonganKasbon float64
	TotalKoreksi        float64
	TotalGajiBruto      float64
	TotalGajiBersih     float64
	JumlahGaji          int
	JumlahDibayar       int
}
