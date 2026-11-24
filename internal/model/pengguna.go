package model

import (
	"errors"
	"time"
)

var (
	ErrUsernameSudahDipakai = errors.New("username sudah terdaftar, silakan pilih username lain")
	ErrKredensialSalah      = errors.New("username atau password salah")
	ErrSesiTidakValid       = errors.New("sesi login tidak ditemukan atau sudah berakhir")
)

type PeranPengguna string

const (
	PeranAdmin   PeranPengguna = "admin"   // Admin Penggajian
	PeranPemilik PeranPengguna = "pemilik" // Pemilik Konveksi (Owner)
)

type Pengguna struct {
	ID           int64
	Username     string
	PasswordHash string
	NamaLengkap  string
	Peran        PeranPengguna
	Aktif        bool
	CreatedAt    time.Time
}

func (p Pengguna) AdalahAdmin() bool {
	return p.Peran == PeranAdmin
}

func (p Pengguna) AdalahPemilik() bool {
	return p.Peran == PeranPemilik
}

// BisaAksesGaji mengembalikan true untuk Admin dan Pemilik.
func (p Pengguna) BisaAksesGaji() bool {
	return p.Peran == PeranAdmin || p.Peran == PeranPemilik
}

func (p Pengguna) LabelPeran() string {
	switch p.Peran {
	case PeranAdmin:
		return "Admin Penggajian"
	case PeranPemilik:
		return "Pemilik Konveksi"
	default:
		return string(p.Peran)
	}
}

type Sesi struct {
	Token      string
	IDPengguna int64
	ExpiredAt  time.Time
	CreatedAt  time.Time
	Pengguna   Pengguna
}

type RingkasanDashboard struct {
	TotalPekerjaAktif         int
	PekerjaHarianAktif        int
	PekerjaBulananAktif       int
	PeriodeTerbuka            int
	AbsensiPerluPeriksa       int
	KasbonAktif               int
	TotalGajiPeriodeBerjalan  string
	TotalSaldoKasbon          string
}

