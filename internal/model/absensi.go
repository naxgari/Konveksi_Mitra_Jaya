package model

import "time"

type Shift struct {
	ID                         int
	Nama, JamMulai, JamSelesai string
}
type PekerjaJadwal struct {
	ID          int
	Nama, Jenis string
}
type JadwalShift struct {
	ID, IDPekerja, IDShift                                     int
	NamaPekerja, JenisPekerja, NamaShift, JamMulai, JamSelesai string
	Tanggal                                                    time.Time
	Absensi                                                    *Absensi
}
type Absensi struct {
	ID, IDJadwal                int
	Status                      string
	JamMasuk, JamKeluar         *time.Time
	MenitTerlambat, MenitLembur int
	Keterangan                  string
}
