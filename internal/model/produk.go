package model

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("data tidak ditemukan")

type Produk struct {
	ID        int
	Name      string
	Price     int
	Stok      int
	Ket       string
	CreatedAt time.Time
}
