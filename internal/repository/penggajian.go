package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/anggyputra/go-class/internal/model"
)

type GajiRepo struct {
	DB *sql.DB
}

type GajiTx struct {
	tx *sql.Tx
}

type DataUpah struct {
	ID         int
	NominalSen int64
}

const kolomGaji = `
	id_gaji, id_pekerja, id_periode,
	gaji_pokok, total_lembur, potongan_telat,
	total_potongan_kasbon, total_koreksi,
	gaji_bruto, gaji_bersih, status_bayar, dibayar_pada
`

func (r *GajiRepo) DaftarPerPeriode(ctx context.Context, idPeriode int) ([]model.Gaji, error) {
	rows, err := r.DB.QueryContext(ctx,
		"SELECT "+kolomGaji+" FROM gaji WHERE id_periode = ? ORDER BY id_pekerja",
		idPeriode,
	)
	if err != nil {
		return nil, fmt.Errorf("daftar gaji periode %d: %w", idPeriode, err)
	}
	defer rows.Close()

	var hasil []model.Gaji

	for rows.Next() {
		var g model.Gaji

		if err := rows.Scan(
			&g.ID,
			&g.IDPekerja,
			&g.IDPeriode,
			&g.GajiPokok,
			&g.TotalLembur,
			&g.PotonganTelat,
			&g.TotalPotonganKasbon,
			&g.TotalKoreksi,
			&g.GajiBruto,
			&g.GajiBersih,
			&g.StatusBayar,
			&g.DibayarPada,
		); err != nil {
			return nil, fmt.Errorf("scan gaji: %w", err)
		}

		hasil = append(hasil, g)
	}

	return hasil, rows.Err()
}

func (r *GajiRepo) CariByID(ctx context.Context, id int) (model.Gaji, error) {
	var g model.Gaji

	err := r.DB.QueryRowContext(ctx,
		"SELECT "+kolomGaji+" FROM gaji WHERE id_gaji = ?", id,
	).Scan(
		&g.ID,
		&g.IDPekerja,
		&g.IDPeriode,
		&g.GajiPokok,
		&g.TotalLembur,
		&g.PotonganTelat,
		&g.TotalPotonganKasbon,
		&g.TotalKoreksi,
		&g.GajiBruto,
		&g.GajiBersih,
		&g.StatusBayar,
		&g.DibayarPada,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return g, model.ErrNotFound
	}
	if err != nil {
		return g, fmt.Errorf("cari gaji %d: %w", id, err)
	}

	return g, nil
}

func (r *GajiRepo) DetailPerGaji(ctx context.Context, idGaji int) ([]model.DetailGaji, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id_detail, id_gaji, id_upah, id_absensi, tanggal,
		       nominal_upah, tarif_per_jam, nilai_pokok,
		       menit_telat, jam_lembur, nilai_lembur, potongan_telat
		FROM detail_gaji
		WHERE id_gaji = ?
		ORDER BY tanggal`, idGaji,
	)
	if err != nil {
		return nil, fmt.Errorf("rincian gaji %d: %w", idGaji, err)
	}
	defer rows.Close()

	var hasil []model.DetailGaji

	for rows.Next() {
		var d model.DetailGaji

		if err := rows.Scan(
			&d.ID,
			&d.IDGaji,
			&d.IDUpah,
			&d.IDAbsensi,
			&d.Tanggal,
			&d.NominalUpah,
			&d.TarifPerJam,
			&d.NilaiPokok,
			&d.MenitTelat,
			&d.JamLembur,
			&d.NilaiLembur,
			&d.PotonganTelat,
		); err != nil {
			return nil, fmt.Errorf("scan rincian gaji: %w", err)
		}

		hasil = append(hasil, d)
	}

	return hasil, rows.Err()
}

func (r *GajiRepo) Transaksi(ctx context.Context, fn func(*GajiTx) error) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := fn(&GajiTx{tx: tx}); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *GajiTx) Pekerja(ctx context.Context, id int) (DataPekerja, error) {
	var p DataPekerja

	err := t.tx.QueryRowContext(ctx, `
		SELECT jenis_pekerja, tanggal_masuk, tanggal_keluar
		FROM pekerja
		WHERE id_pekerja = ?
		FOR UPDATE`, id,
	).Scan(
		&p.Jenis,
		&p.Masuk,
		&p.Keluar,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return p, model.ErrNotFound
	}
	if err != nil {
		return p, fmt.Errorf("ambil pekerja untuk penggajian: %w", err)
	}

	return p, nil
}

func (t *GajiTx) Periode(ctx context.Context, id int) (model.PeriodeGaji, error) {
	var p model.PeriodeGaji

	err := t.tx.QueryRowContext(ctx, `
		SELECT id_periode, jenis_periode, tanggal_awal,
		       tanggal_akhir, tanggal_bayar_rencana, status
		FROM periode_gaji
		WHERE id_periode = ?
		FOR UPDATE`, id,
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
		return p, fmt.Errorf("ambil periode untuk penggajian: %w", err)
	}

	return p, nil
}

func (t *GajiTx) UpahBerlaku(
	ctx context.Context,
	idPekerja int,
	tanggal time.Time,
) (DataUpah, error) {
	var upah DataUpah

	err := t.tx.QueryRowContext(ctx, `
		SELECT id_upah, CAST(nominal * 100 AS SIGNED)
		FROM riwayat_upah
		WHERE id_pekerja = ? AND berlaku_mulai <= ?
		ORDER BY berlaku_mulai DESC
		LIMIT 1
		FOR UPDATE`,
		idPekerja,
		tanggal.Format("2006-01-02"),
	).Scan(
		&upah.ID,
		&upah.NominalSen,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return upah, model.ErrNotFound
	}
	if err != nil {
		return upah, fmt.Errorf("ambil tarif upah: %w", err)
	}

	return upah, nil
}

func (t *GajiTx) JadwalAbsensi(
	ctx context.Context,
	idPekerja int,
	mulai, akhir time.Time,
) ([]model.JadwalShift, error) {
	rows, err := t.tx.QueryContext(ctx,
		jadwalSelect+`
		WHERE j.id_pekerja = ? AND j.tanggal BETWEEN ? AND ?
		ORDER BY j.tanggal
		FOR UPDATE`,
		idPekerja,
		mulai.Format("2006-01-02"),
		akhir.Format("2006-01-02"),
	)
	if err != nil {
		return nil, fmt.Errorf("ambil jadwal dan absensi: %w", err)
	}
	defer rows.Close()

	var hasil []model.JadwalShift

	for rows.Next() {
		jadwal, err := scanJadwal(rows)
		if err != nil {
			return nil, fmt.Errorf("scan jadwal penggajian: %w", err)
		}

		hasil = append(hasil, jadwal)
	}

	return hasil, rows.Err()
}

func (t *GajiTx) GajiPekerja(
	ctx context.Context,
	idPekerja, idPeriode int,
) (model.Gaji, error) {
	var g model.Gaji

	err := t.tx.QueryRowContext(ctx,
		"SELECT "+kolomGaji+`
		FROM gaji
		WHERE id_pekerja = ? AND id_periode = ?
		FOR UPDATE`,
		idPekerja, idPeriode,
	).Scan(
		&g.ID,
		&g.IDPekerja,
		&g.IDPeriode,
		&g.GajiPokok,
		&g.TotalLembur,
		&g.PotonganTelat,
		&g.TotalPotonganKasbon,
		&g.TotalKoreksi,
		&g.GajiBruto,
		&g.GajiBersih,
		&g.StatusBayar,
		&g.DibayarPada,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return g, model.ErrNotFound
	}
	if err != nil {
		return g, fmt.Errorf("ambil gaji pekerja: %w", err)
	}

	return g, nil
}

func (t *GajiTx) TambahGaji(ctx context.Context, g model.Gaji) (int64, error) {
	hasil, err := t.tx.ExecContext(ctx, `
		INSERT INTO gaji (
			id_pekerja, id_periode,
			gaji_pokok, total_lembur, potongan_telat,
			total_potongan_kasbon, total_koreksi,
			gaji_bruto, gaji_bersih, status_bayar
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'belum')`,
		g.IDPekerja,
		g.IDPeriode,
		g.GajiPokok,
		g.TotalLembur,
		g.PotonganTelat,
		g.TotalPotonganKasbon,
		g.TotalKoreksi,
		g.GajiBruto,
		g.GajiBersih,
	)
	if err != nil {
		return 0, fmt.Errorf("tambah draf gaji: %w", err)
	}

	return hasil.LastInsertId()
}

func (t *GajiTx) UbahDraf(ctx context.Context, g model.Gaji) error {
	_, err := t.tx.ExecContext(ctx, `
		UPDATE gaji
		SET gaji_pokok = ?,
		    total_lembur = ?,
		    potongan_telat = ?,
		    total_potongan_kasbon = ?,
		    total_koreksi = ?,
		    gaji_bruto = ?,
		    gaji_bersih = ?
		WHERE id_gaji = ? AND status_bayar = 'belum'`,
		g.GajiPokok,
		g.TotalLembur,
		g.PotonganTelat,
		g.TotalPotonganKasbon,
		g.TotalKoreksi,
		g.GajiBruto,
		g.GajiBersih,
		g.ID,
	)
	if err != nil {
		return fmt.Errorf("ubah draf gaji: %w", err)
	}

	return nil
}

func (t *GajiTx) HapusDetailDraf(ctx context.Context, idGaji int) error {
	_, err := t.tx.ExecContext(ctx, `
		DELETE d
		FROM detail_gaji AS d
		JOIN gaji AS g ON g.id_gaji = d.id_gaji
		WHERE g.id_gaji = ? AND g.status_bayar = 'belum'`,
		idGaji,
	)
	if err != nil {
		return fmt.Errorf("hapus rincian draf gaji: %w", err)
	}

	return nil
}

func (t *GajiTx) TambahDetail(ctx context.Context, d model.DetailGaji) error {
	_, err := t.tx.ExecContext(ctx, `
		INSERT INTO detail_gaji (
			id_gaji, id_upah, id_absensi, tanggal,
			nominal_upah, tarif_per_jam, nilai_pokok,
			menit_telat, jam_lembur, nilai_lembur, potongan_telat
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		d.IDGaji,
		d.IDUpah,
		d.IDAbsensi,
		d.Tanggal.Format("2006-01-02"),
		d.NominalUpah,
		d.TarifPerJam,
		d.NilaiPokok,
		d.MenitTelat,
		d.JamLembur,
		d.NilaiLembur,
		d.PotonganTelat,
	)
	if err != nil {
		return fmt.Errorf("tambah rincian gaji: %w", err)
	}

	return nil
}

func (t *GajiTx) PerluPenyesuaianPembayaran(
	ctx context.Context,
	idPekerja, idGaji int,
	periode model.PeriodeGaji,
) (bool, error) {
	var perlu bool

	err := t.tx.QueryRowContext(ctx, `
		SELECT
			EXISTS (
				SELECT 1 FROM kasbon k
				WHERE k.id_pekerja = ?
				  AND k.tanggal <= ?
				  AND k.jumlah > COALESCE((
					SELECT SUM(pk.jumlah)
					FROM potongan_kasbon pk
					JOIN gaji g ON g.id_gaji = pk.id_gaji
					WHERE pk.id_kasbon = k.id_kasbon
					  AND g.status_bayar = 'dibayar'
				  ), 0)
			)
			OR EXISTS (
				SELECT 1 FROM potongan_kasbon
				WHERE id_gaji = ?
			)
			OR EXISTS (
				SELECT 1
				FROM koreksi_gaji k
				JOIN gaji asal ON asal.id_gaji = k.id_gaji_asal
				JOIN periode_gaji p ON p.id_periode = asal.id_periode
				WHERE k.id_gaji_tujuan = ?
				   OR (
					asal.id_pekerja = ?
					AND k.status = 'disetujui'
					AND p.tanggal_akhir < ?
				   )
			)`,
		idPekerja,
		periode.TanggalAkhir.Format("2006-01-02"),
		idGaji,
		idGaji,
		idPekerja,
		periode.TanggalAwal.Format("2006-01-02"),
	).Scan(&perlu)
	if err != nil {
		return false, fmt.Errorf("periksa penyesuaian pembayaran: %w", err)
	}

	return perlu, nil
}

func (t *GajiTx) CatatPembayaran(ctx context.Context, idGaji int) error {
	hasil, err := t.tx.ExecContext(ctx, `
		UPDATE gaji AS g
		JOIN periode_gaji AS p ON p.id_periode = g.id_periode
		SET g.status_bayar = 'dibayar',
		    g.dibayar_pada = CURRENT_TIMESTAMP
		WHERE g.id_gaji = ?
		  AND g.status_bayar = 'belum'
		  AND g.dibayar_pada IS NULL
		  AND p.status = 'terbuka'
		  AND g.gaji_bersih >= 0`,
		idGaji,
	)
	if err != nil {
		return fmt.Errorf("catat pembayaran gaji: %w", err)
	}

	jumlah, err := hasil.RowsAffected()
	if err != nil {
		return fmt.Errorf("periksa hasil pembayaran gaji: %w", err)
	}
	if jumlah == 0 {
		return &AturanError{
			Pesan: "Gaji tidak dapat dibayar; periksa status gaji, periode, dan gaji bersih",
		}
	}

	return nil
}


func (r *GajiRepo) PekerjaPerPeriode(
	ctx context.Context,
	idPeriode int,
) ([]model.PekerjaGaji, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT p.id_pekerja, p.nama
		FROM pekerja p
		JOIN periode_gaji pg ON pg.id_periode = ?
		LEFT JOIN gaji g
			ON g.id_pekerja = p.id_pekerja
			AND g.id_periode = pg.id_periode
		WHERE (
			(pg.jenis_periode = 'mingguan' AND p.jenis_pekerja = 'harian')
			OR
			(pg.jenis_periode = 'bulanan' AND p.jenis_pekerja = 'bulanan')
		)
		AND p.tanggal_masuk <= pg.tanggal_akhir
		AND (
			p.tanggal_keluar IS NULL
			OR p.tanggal_keluar >= pg.tanggal_awal
		)
		AND (g.id_gaji IS NULL OR g.status_bayar = 'belum')
		ORDER BY p.nama, p.id_pekerja`, idPeriode,
	)
	if err != nil {
		return nil, fmt.Errorf("daftar pekerja periode %d: %w", idPeriode, err)
	}
	defer rows.Close()

	var hasil []model.PekerjaGaji

	for rows.Next() {
		var p model.PekerjaGaji

		if err := rows.Scan(&p.ID, &p.Nama); err != nil {
			return nil, fmt.Errorf("scan pekerja penggajian: %w", err)
		}

		hasil = append(hasil, p)
	}

	return hasil, rows.Err()
}