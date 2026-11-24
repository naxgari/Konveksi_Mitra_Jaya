package model

import "time"

// RiwayatUpah menyimpan perubahan tarif upah pekerja.
type RiwayatUpah struct {
	ID           int
	IDPekerja    int
	BerlakuMulai time.Time
	Nominal      string
	Keterangan   *string
	CreatedAt    time.Time
}
