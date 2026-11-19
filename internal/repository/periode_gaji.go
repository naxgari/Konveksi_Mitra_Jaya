package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/anggyputra/go-class/internal/model"
)

type PeriodeGajiRepo struct {
	DB *sql.DB
}

type PeriodeGajiTx struct {
	tx *sql.Tx
}

func (r *PeriodeGajiRepo) Semua(ctx context.Context) ([]model.PeriodeGaji, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id_periode, jenis_periode, tanggal_awal, tanggal_akhir,
		       tanggal_bayar_rencana, status
		FROM periode_gaji
		ORDER BY tanggal_awal DESC, id_periode DESC`)
	if err != nil {
		return nil, fmt.Errorf("daftar periode gaji: %w", err)
	}
	defer rows.Close()

	var hasil []model.PeriodeGaji

	for rows.Next() {
		var p model.PeriodeGaji
		if err := rows.Scan(
			&p.ID,
			&p.Jenis,
			&p.TanggalAwal,
			&p.TanggalAkhir,
			&p.TanggalBayarRencana,
			&p.Status,
		); err != nil {
			return nil, fmt.Errorf("scan periode gaji: %w", err)
		}

		hasil = append(hasil, p)
	}

	return hasil, rows.Err()
}

func (r *PeriodeGajiRepo) CariByID(ctx context.Context, id int) (model.PeriodeGaji, error) {
	var p model.PeriodeGaji

	err := r.DB.QueryRowContext(ctx, `
		SELECT id_periode, jenis_periode, tanggal_awal, tanggal_akhir,
		       tanggal_bayar_rencana, status
		FROM periode_gaji
		WHERE id_periode = ?`, id,
	).Scan(
		&p.ID,
		&p.Jenis,
		&p.TanggalAwal,
		&p.TanggalAkhir,
		&p.TanggalBayarRencana,
		&p.Status,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return p, model.ErrNotFound
	}
	if err != nil {
		return p, fmt.Errorf("cari periode gaji %d: %w", id, err)
	}

	return p, nil
}

func (r *PeriodeGajiRepo) Transaksi(
	ctx context.Context,
	fn func(*PeriodeGajiTx) error,
) error {
	tx, err := r.DB.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := fn(&PeriodeGajiTx{tx: tx}); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *PeriodeGajiTx) DaftarUntukValidasi(ctx context.Context) ([]model.PeriodeGaji, error) {
	rows, err := t.tx.QueryContext(ctx, `
		SELECT jenis_periode, tanggal_awal, tanggal_akhir
		FROM periode_gaji
		ORDER BY id_periode
		FOR UPDATE`)
	if err != nil {
		return nil, fmt.Errorf("baca periode untuk validasi: %w", err)
	}
	defer rows.Close()

	var hasil []model.PeriodeGaji

	for rows.Next() {
		var p model.PeriodeGaji
		if err := rows.Scan(
			&p.Jenis,
			&p.TanggalAwal,
			&p.TanggalAkhir,
		); err != nil {
			return nil, fmt.Errorf("scan periode untuk validasi: %w", err)
		}

		hasil = append(hasil, p)
	}

	return hasil, rows.Err()
}

func (t *PeriodeGajiTx) StatusPeriode(ctx context.Context, id int) (string, error) {
	var status string

	err := t.tx.QueryRowContext(ctx, `
		SELECT status FROM periode_gaji
		WHERE id_periode = ?
		FOR UPDATE`, id,
	).Scan(&status)

	if errors.Is(err, sql.ErrNoRows) {
		return "", model.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("ambil status periode: %w", err)
	}

	return status, nil
}

func (t *PeriodeGajiTx) StatusGaji(ctx context.Context, idPeriode int) ([]string, error) {
	rows, err := t.tx.QueryContext(ctx, `
		SELECT status_bayar FROM gaji
		WHERE id_periode = ?
		ORDER BY id_gaji
		FOR UPDATE`, idPeriode)
	if err != nil {
		return nil, fmt.Errorf("ambil status gaji periode: %w", err)
	}
	defer rows.Close()

	var daftar []string

	for rows.Next() {
		var status string
		if err := rows.Scan(&status); err != nil {
			return nil, fmt.Errorf("baca status gaji: %w", err)
		}
		daftar = append(daftar, status)
	}

	return daftar, rows.Err()
}

func (t *PeriodeGajiTx) Tambah(ctx context.Context, p model.PeriodeGaji) (int64, error) {
	hasil, err := t.tx.ExecContext(ctx, `
		INSERT INTO periode_gaji (
			jenis_periode,
			tanggal_awal,
			tanggal_akhir,
			tanggal_bayar_rencana,
			status
		)
		VALUES (?, ?, ?, ?, 'terbuka')`,
		p.Jenis,
		p.TanggalAwal.Format("2006-01-02"),
		p.TanggalAkhir.Format("2006-01-02"),
		p.TanggalBayarRencana.Format("2006-01-02"),
	)
	if err != nil {
		return 0, fmt.Errorf("tambah periode gaji: %w", err)
	}

	return hasil.LastInsertId()
}

func (t *PeriodeGajiTx) Tutup(
	ctx context.Context,
	idPeriode int,
	idPengguna int64,
) error {
	hasil, err := t.tx.ExecContext(ctx, `
		UPDATE periode_gaji AS p
		JOIN pengguna AS u ON u.id_pengguna = ?
		SET p.status = 'tertutup',
		    p.ditutup_oleh = u.id_pengguna,
		    p.ditutup_pada = CURRENT_TIMESTAMP
		WHERE p.id_periode = ?
		  AND p.status = 'terbuka'
		  AND p.ditutup_pada IS NULL
		  AND u.peran = 'pemilik'
		  AND u.aktif = 1
		  AND NOT EXISTS (
			SELECT 1 FROM gaji g
			WHERE g.id_periode = p.id_periode
			  AND g.status_bayar != 'dibayar'
		  )`,
		idPengguna,
		idPeriode,
	)
	if err != nil {
		return fmt.Errorf("tutup periode gaji: %w", err)
	}

	jumlah, err := hasil.RowsAffected()
	if err != nil {
		return fmt.Errorf("periksa hasil penutupan periode: %w", err)
	}
	if jumlah == 0 {
		return &AturanError{
			Pesan: "Periode tidak dapat ditutup; periksa akun Pemilik dan status pembayaran",
		}
	}

	return nil
}
