package model

import "time"

// Pekerja merepresentasikan data pekerja konveksi.
type Pekerja struct {
	ID            int
	Nama          string
	JenisPekerja  string
	Jabatan       string
	Kontak        *string
	TanggalMasuk  time.Time
	TanggalKeluar *time.Time
	CreatedAt     time.Time
	UpdatedAt     *time.Time
}
