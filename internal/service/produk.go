package service

import (
	"context"
	"strings"
	"time"

	"github.com/anggyputra/go-class/internal/model"
	"github.com/anggyputra/go-class/internal/repository"
)

type ValidasiError struct {
	Pesan string
}

func (e *ValidasiError) Error() string { return e.Pesan }

type ProdukService struct {
	Repo *repository.ProdukRepo
}

func (s *ProdukService) Daftar(ctx context.Context) ([]model.Produk, error) {
	return s.Repo.Semua(ctx)
}

func (s *ProdukService) Detail(ctx context.Context, id int) (model.Produk, error) {
	return s.Repo.CariByID(ctx, id)
}

func (s *ProdukService) Buat(ctx context.Context, p model.Produk) (int64, error) {
	if err := validasi(p); err != nil {
		return 0, err
	}
	p.CreatedAt = time.Now()
	return s.Repo.Tambah(ctx, p)
}

func (s *ProdukService) Ubah(ctx context.Context, p model.Produk) error {
	if err := validasi(p); err != nil {
		return err
	}
	return s.Repo.Ubah(ctx, p)
}

func (s *ProdukService) Hapus(ctx context.Context, id int) error {
	return s.Repo.Hapus(ctx, id)
}

func validasi(p model.Produk) error {
	if strings.TrimSpace(p.Name) == "" {
		return &ValidasiError{"Nama wajib diisi"}
	}
	if p.Price < 0 {
		return &ValidasiError{"Harga tidak boleh negatif"}
	}
	if p.Stok < 0 {
		return &ValidasiError{"Stok tidak boleh negatif"}
	}
	return nil
}
