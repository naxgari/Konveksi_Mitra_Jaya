package model

import "time"

type Produk struct {
	ID        int
	Name      string
	Price     int
	Stok      int
	Ket       string
	CreatedAt time.Time
}
