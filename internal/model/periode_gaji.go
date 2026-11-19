package model

import "time"

type PeriodeGaji struct {
	ID                  int
	Jenis               string
	TanggalAwal         time.Time
	TanggalAkhir        time.Time
	TanggalBayarRencana time.Time
	Status              string
}

