package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/anggyputra/go-class/internal/model"
)

type ProdukRepo struct {
	DB *sql.DB
}

const kolomProduk = "id, nama, harga, stok, coalesce(keterangan,'') as keterangan, created_at"

func (r *ProdukRepo) Semua(ctx context.Context) ([]model.Produk, error) {
	rows, err := r.DB.QueryContext(ctx,
		"SELECT "+kolomProduk+" FROM produk WHERE deleted_at IS NULL ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("daftar produk: %w", err)
	}
	defer rows.Close()

	var hasil []model.Produk
	for rows.Next() {
		var p model.Produk
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Stok, &p.Ket, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan produk: %w", err)
		}
		hasil = append(hasil, p)
	}
	return hasil, rows.Err()
}

func (r *ProdukRepo) CariByID(ctx context.Context, id int) (model.Produk, error) {
	var p model.Produk
	err := r.DB.QueryRowContext(ctx,
		"SELECT "+kolomProduk+" FROM produk WHERE id = ? AND deleted_at IS NULL", id,
	).Scan(&p.ID, &p.Name, &p.Price, &p.Stok, &p.Ket, &p.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return p, model.ErrNotFound
	}
	if err != nil {
		return p, fmt.Errorf("cari produk %d: %w", id, err)
	}
	return p, nil
}

func (r *ProdukRepo) Tambah(ctx context.Context, p model.Produk) (int64, error) {
	res, err := r.DB.ExecContext(ctx,
		"INSERT INTO produk (nama, harga, stok, keterangan, created_at) VALUES (?, ?, ?, ?, ?)",
		p.Name, p.Price, p.Stok, p.Ket, p.CreatedAt,
	)
	if err != nil {
		return 0, fmt.Errorf("tambah produk: %w", err)
	}
	return res.LastInsertId()
}

func (r *ProdukRepo) Ubah(ctx context.Context, p model.Produk) error {
	res, err := r.DB.ExecContext(ctx,
		"UPDATE produk SET nama=?, harga=?, stok=?, keterangan=?, updated_at=NOW() WHERE id=? AND deleted_at IS NULL",
		p.Name, p.Price, p.Stok, p.Ket, p.ID,
	)
	if err != nil {
		return fmt.Errorf("ubah produk %d: %w", p.ID, err)
	}
	return pastikanAdaBaris(res)
}

func (r *ProdukRepo) Hapus(ctx context.Context, id int) error {
	res, err := r.DB.ExecContext(ctx,
		"UPDATE produk SET deleted_at = NOW() WHERE id = ? AND deleted_at IS NULL", id,
	)
	if err != nil {
		return fmt.Errorf("hapus produk %d: %w", id, err)
	}
	return pastikanAdaBaris(res)
}

func pastikanAdaBaris(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}
