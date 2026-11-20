package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/anggyputra/go-class/internal/model"
)

type PekerjaRepo struct {
	DB *sql.DB
}

const kolomPekerja = "id_pekerja, nama, jenis_pekerja, jabatan, kontak, tanggal_masuk, tanggal_keluar, created_at, updated_at"

// Semua mengambil seluruh data pekerja.
func (r *PekerjaRepo) Semua(ctx context.Context) ([]model.Pekerja, error) {
	rows, err := r.DB.QueryContext(ctx,
		"SELECT "+kolomPekerja+" FROM pekerja ORDER BY id_pekerja")
	if err != nil {
		return nil, fmt.Errorf("daftar pekerja: %w", err)
	}
	defer rows.Close()

	var hasil []model.Pekerja

	for rows.Next() {
		var p model.Pekerja

		err := rows.Scan(
			&p.ID,
			&p.Nama,
			&p.JenisPekerja,
			&p.Jabatan,
			&p.Kontak,
			&p.TanggalMasuk,
			&p.TanggalKeluar,
			&p.CreatedAt,
			&p.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("scan pekerja: %w", err)
		}

		hasil = append(hasil, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterasi pekerja: %w", err)
	}

	return hasil, nil
}

// CariByID mengambil data pekerja berdasarkan ID.
func (r *PekerjaRepo) CariByID(ctx context.Context, id int) (model.Pekerja, error) {
	var p model.Pekerja

	err := r.DB.QueryRowContext(ctx,
		"SELECT "+kolomPekerja+" FROM pekerja WHERE id_pekerja = ?",
		id,
	).Scan(
		&p.ID,
		&p.Nama,
		&p.JenisPekerja,
		&p.Jabatan,
		&p.Kontak,
		&p.TanggalMasuk,
		&p.TanggalKeluar,
		&p.CreatedAt,
		&p.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return p, model.ErrNotFound
	}

	if err != nil {
		return p, fmt.Errorf("cari pekerja %d: %w", id, err)
	}

	return p, nil
}

// Tambah menyimpan data pekerja baru ke database.
func (r *PekerjaRepo) Tambah(ctx context.Context, p model.Pekerja) (int64, error) {
	res, err := r.DB.ExecContext(ctx,
		`INSERT INTO pekerja
		(nama, jenis_pekerja, jabatan, kontak, tanggal_masuk, tanggal_keluar)
		VALUES (?, ?, ?, ?, ?, ?)`,
		p.Nama,
		p.JenisPekerja,
		p.Jabatan,
		p.Kontak,
		p.TanggalMasuk,
		p.TanggalKeluar,
	)

	if err != nil {
		return 0, fmt.Errorf("tambah pekerja: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("ambil ID pekerja baru: %w", err)
	}

	return id, nil
}

// Ubah memperbarui data pekerja berdasarkan ID.
func (r *PekerjaRepo) Ubah(ctx context.Context, p model.Pekerja) error {
	res, err := r.DB.ExecContext(ctx,
		`UPDATE pekerja
		SET nama = ?,
		    jenis_pekerja = ?,
		    jabatan = ?,
		    kontak = ?,
		    tanggal_masuk = ?,
		    tanggal_keluar = ?,
		    updated_at = NOW()
		WHERE id_pekerja = ?`,
		p.Nama,
		p.JenisPekerja,
		p.Jabatan,
		p.Kontak,
		p.TanggalMasuk,
		p.TanggalKeluar,
		p.ID,
	)

	if err != nil {
		return fmt.Errorf("ubah pekerja %d: %w", p.ID, err)
	}

	jumlah, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("periksa perubahan pekerja: %w", err)
	}

	if jumlah == 0 {
		var ada bool
		err := r.DB.QueryRowContext(ctx,
			"SELECT EXISTS(SELECT 1 FROM pekerja WHERE id_pekerja = ?)",
			p.ID,
		).Scan(&ada)
		if err != nil {
			return fmt.Errorf("periksa pekerja %d: %w", p.ID, err)
		}
		if !ada {
			return model.ErrNotFound
		}
	}

	return nil
}

// Nonaktifkan mencatat tanggal keluar pekerja tanpa menghapus datanya.
func (r *PekerjaRepo) Nonaktifkan(ctx context.Context, id int, tanggalKeluar time.Time) error {
	if tanggalKeluar.IsZero() {
		return fmt.Errorf("tanggal keluar wajib diisi")
	}

	// Pastikan pekerja ditemukan.
	p, err := r.CariByID(ctx, id)
	if err != nil {
		return err
	}

	// Pekerja yang sudah keluar tidak boleh dinonaktifkan lagi.
	if p.TanggalKeluar != nil {
		return fmt.Errorf("pekerja %d sudah nonaktif", id)
	}

	// Tanggal keluar tidak boleh lebih awal dari tanggal masuk.
	if tanggalKeluar.Format("2006-01-02") < p.TanggalMasuk.Format("2006-01-02") {
		return fmt.Errorf("tanggal keluar tidak boleh sebelum tanggal masuk")
	}

	res, err := r.DB.ExecContext(ctx,
		`UPDATE pekerja
		SET tanggal_keluar = ?,
		    updated_at = NOW()
		WHERE id_pekerja = ? AND tanggal_keluar IS NULL`,
		tanggalKeluar,
		id,
	)

	if err != nil {
		return fmt.Errorf("nonaktifkan pekerja %d: %w", id, err)
	}

	jumlah, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("periksa nonaktifkan pekerja: %w", err)
	}

	if jumlah == 0 {
		return fmt.Errorf("pekerja %d tidak dapat dinonaktifkan karena datanya telah berubah", id)
	}

	return nil
}
