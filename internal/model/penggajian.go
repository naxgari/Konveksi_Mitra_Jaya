package model

import "time"

type Gaji struct {
	ID                  int
	IDPekerja           int
	NamaPekerja         string
	JenisPekerja        string
	IDPeriode           int
	GajiPokok           string
	TotalLembur         string
	PotonganTelat       string
	TotalPotonganKasbon string
	TotalKoreksi        string
	GajiBruto           string
	GajiBersih          string
	StatusBayar         string
	DibayarPada         *time.Time
}

type DetailGaji struct {
	ID            int
	IDGaji        int
	IDUpah        int
	IDAbsensi     *int
	Tanggal       time.Time
	NominalUpah   string
	TarifPerJam   string
	NilaiPokok    string
	MenitTelat    int
	JamLembur     string
	NilaiLembur   string
	PotonganTelat string
}

type PekerjaGaji struct {
	ID   int
	Nama string
}