package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/anggyputra/go-class/internal/model"
)

// RiwayatUpahRepo menangani akses data riwayat upah.
type RiwayatUpahRepo struct {
	DB *sql.DB
}

const kolomRiwayatUpah = "id_upah, id_pekerja, berlaku_mulai, nominal, keterangan, created_at"

// SemuaByPekerja mengambil riwayat upah seorang pekerja.
func (r *RiwayatUpahRepo) SemuaByPekerja(
	ctx context.Context,
	idPekerja int,
) ([]model.RiwayatUpah, error) {

	rows, err := r.DB.QueryContext(ctx,
		`SELECT `+kolomRiwayatUpah+`
		FROM riwayat_upah
		WHERE id_pekerja = ?
		ORDER BY berlaku_mulai DESC, id_upah DESC`,
		idPekerja,
	)
	if err != nil {
		return nil, fmt.Errorf("daftar riwayat upah: %w", err)
	}
	defer rows.Close()

	var hasil []model.RiwayatUpah

	for rows.Next() {
		var u model.RiwayatUpah

		err := rows.Scan(
			&u.ID,
			&u.IDPekerja,
			&u.BerlakuMulai,
			&u.Nominal,
			&u.Keterangan,
			&u.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan riwayat upah: %w", err)
		}

		hasil = append(hasil, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterasi riwayat upah: %w", err)
	}

	return hasil, nil
}

// Tambah mencatat tarif upah baru.
func (r *RiwayatUpahRepo) Tambah(
	ctx context.Context,
	u model.RiwayatUpah,
) (int64, error) {

	res, err := r.DB.ExecContext(ctx,
		`INSERT INTO riwayat_upah
		(id_pekerja, berlaku_mulai, nominal, keterangan)
		VALUES (?, ?, ?, ?)`,
		u.IDPekerja,
		u.BerlakuMulai,
		u.Nominal,
		u.Keterangan,
	)

	if err != nil {
		return 0, fmt.Errorf("tambah riwayat upah: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("ambil ID riwayat upah: %w", err)
	}

	return id, nil
}

// TarifBerlaku mencari tarif yang berlaku pada tanggal tertentu.
func (r *RiwayatUpahRepo) TarifBerlaku(
	ctx context.Context,
	idPekerja int,
	tanggal time.Time,
) (model.RiwayatUpah, error) {

	var u model.RiwayatUpah

	err := r.DB.QueryRowContext(ctx,
		`SELECT `+kolomRiwayatUpah+`
		FROM riwayat_upah
		WHERE id_pekerja = ?
		  AND berlaku_mulai <= ?
		ORDER BY berlaku_mulai DESC, id_upah DESC
		LIMIT 1`,
		idPekerja,
		tanggal,
	).Scan(
		&u.ID,
		&u.IDPekerja,
		&u.BerlakuMulai,
		&u.Nominal,
		&u.Keterangan,
		&u.CreatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return u, model.ErrNotFound
	}

	if err != nil {
		return u, fmt.Errorf("cari tarif berlaku: %w", err)
	}

	return u, nil
}
