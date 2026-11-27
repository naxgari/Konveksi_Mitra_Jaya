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

func formatTanggalPtr(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return t.Format("2006-01-02")
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
		p.TanggalMasuk.Format("2006-01-02"),
		formatTanggalPtr(p.TanggalKeluar),
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

// TambahDenganUpahAwal menyimpan data pekerja baru beserta baris awal riwayat_upah dalam satu transaksi (AB 02).
func (r *PekerjaRepo) TambahDenganUpahAwal(
	ctx context.Context,
	p model.Pekerja,
	nominalUpah string,
	keterangan *string,
) (int64, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("mulai transaksi tambah pekerja: %w", err)
	}
	defer tx.Rollback()

	tanggalMasuk := p.TanggalMasuk.Format("2006-01-02")
	res, err := tx.ExecContext(ctx,
		`INSERT INTO pekerja
		(nama, jenis_pekerja, jabatan, kontak, tanggal_masuk, tanggal_keluar)
		VALUES (?, ?, ?, ?, ?, ?)`,
		p.Nama,
		p.JenisPekerja,
		p.Jabatan,
		p.Kontak,
		tanggalMasuk,
		formatTanggalPtr(p.TanggalKeluar),
	)
	if err != nil {
		return 0, fmt.Errorf("tambah pekerja: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("ambil ID pekerja baru: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO riwayat_upah (id_pekerja, berlaku_mulai, nominal, keterangan)
		VALUES (?, ?, ?, ?)`,
		id,
		tanggalMasuk,
		nominalUpah,
		keterangan,
	); err != nil {
		return 0, fmt.Errorf("tambah riwayat upah awal: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit tambah pekerja dan upah awal: %w", err)
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
		p.TanggalMasuk.Format("2006-01-02"),
		formatTanggalPtr(p.TanggalKeluar),
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
	res, err := r.DB.ExecContext(ctx,
		`UPDATE pekerja
		SET tanggal_keluar = ?,
		    updated_at = NOW()
		WHERE id_pekerja = ? AND tanggal_keluar IS NULL`,
		tanggalKeluar.Format("2006-01-02"),
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
		return &AturanError{Pesan: "Pekerja sudah dinonaktifkan atau datanya telah berubah"}
	}

	return nil
}
