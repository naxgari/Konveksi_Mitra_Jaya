package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
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

type KasbonSaldo struct {
	ID       int
	Tanggal  time.Time
	SaldoSen int64
}

type KoreksiTerpilih struct {
	ID       int
	NilaiSen int64
}

func parseDesimalSen(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	negatif := false
	if strings.HasPrefix(s, "-") {
		negatif = true
		s = strings.TrimPrefix(s, "-")
	} else if strings.HasPrefix(s, "+") {
		s = strings.TrimPrefix(s, "+")
	}
	bagian := strings.SplitN(s, ".", 2)
	rupiahVal, err := strconv.ParseInt(bagian[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse nominal rupiah %q: %w", s, err)
	}
	var senVal int64
	if len(bagian) == 2 {
		des := bagian[1]
		if len(des) == 1 {
			des += "0"
		} else if len(des) > 2 {
			des = des[:2]
		}
		senVal, err = strconv.ParseInt(des, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse sen %q: %w", s, err)
		}
	}
	total := rupiahVal*100 + senVal
	if negatif {
		total = -total
	}
	return total, nil
}

const kolomGaji = `
	id_gaji, id_pekerja, id_periode,
	gaji_pokok, total_lembur, potongan_telat,
	total_potongan_kasbon, total_koreksi,
	gaji_bruto, gaji_bersih, status_bayar, dibayar_pada
`

func (r *GajiRepo) DaftarPerPeriode(ctx context.Context, idPeriode int) ([]model.Gaji, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT g.id_gaji, g.id_pekerja, p.nama, p.jenis_pekerja, g.id_periode,
		       g.gaji_pokok, g.total_lembur, g.potongan_telat,
		       g.total_potongan_kasbon, g.total_koreksi,
		       g.gaji_bruto, g.gaji_bersih, g.status_bayar, g.dibayar_pada
		FROM gaji g
		JOIN pekerja p ON p.id_pekerja = g.id_pekerja
		WHERE g.id_periode = ?
		ORDER BY p.nama, g.id_pekerja`,
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
			&g.NamaPekerja,
			&g.JenisPekerja,
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

	err := r.DB.QueryRowContext(ctx, `
		SELECT g.id_gaji, g.id_pekerja, p.nama, p.jenis_pekerja, g.id_periode,
		       g.gaji_pokok, g.total_lembur, g.potongan_telat,
		       g.total_potongan_kasbon, g.total_koreksi,
		       g.gaji_bruto, g.gaji_bersih, g.status_bayar, g.dibayar_pada
		FROM gaji g
		JOIN pekerja p ON p.id_pekerja = g.id_pekerja
		WHERE g.id_gaji = ?`, id,
	).Scan(
		&g.ID,
		&g.IDPekerja,
		&g.NamaPekerja,
		&g.JenisPekerja,
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

func (t *GajiTx) KoreksiDisetujui(
	ctx context.Context,
	idPekerja int,
	periode model.PeriodeGaji,
) ([]KoreksiTerpilih, error) {
	awalStr := periode.TanggalAwal.Format("2006-01-02")
	rows, err := t.tx.QueryContext(ctx, `
		SELECT kg.id_koreksi, kg.nilai_selisih
		FROM koreksi_gaji kg
		JOIN gaji ga ON ga.id_gaji = kg.id_gaji_asal
		JOIN periode_gaji pa ON pa.id_periode = ga.id_periode
		WHERE ga.id_pekerja = ?
		  AND kg.status = 'disetujui'
		  AND kg.id_gaji_tujuan IS NULL
		  AND pa.jenis_periode = ?
		  AND pa.tanggal_awal < ?
		  AND NOT EXISTS (
			SELECT 1
			FROM gaji g2
			JOIN periode_gaji p2 ON p2.id_periode = g2.id_periode
			WHERE g2.id_pekerja = ga.id_pekerja
			  AND p2.jenis_periode = pa.jenis_periode
			  AND p2.status = 'terbuka'
			  AND g2.status_bayar = 'belum'
			  AND p2.tanggal_awal > pa.tanggal_awal
			  AND p2.tanggal_awal < ?
		  )
		ORDER BY kg.tanggal ASC, kg.id_koreksi ASC
		FOR UPDATE`,
		idPekerja,
		periode.Jenis,
		awalStr,
		awalStr,
	)
	if err != nil {
		return nil, fmt.Errorf("ambil koreksi disetujui: %w", err)
	}
	defer rows.Close()

	var hasil []KoreksiTerpilih
	for rows.Next() {
		var id int
		var nilaiStr string
		if err := rows.Scan(&id, &nilaiStr); err != nil {
			return nil, fmt.Errorf("scan koreksi disetujui: %w", err)
		}
		sen, err := parseDesimalSen(nilaiStr)
		if err != nil {
			return nil, err
		}
		hasil = append(hasil, KoreksiTerpilih{ID: id, NilaiSen: sen})
	}
	return hasil, rows.Err()
}

func (t *GajiTx) KasbonBersaldo(
	ctx context.Context,
	idPekerja int,
	tanggalAkhir time.Time,
) ([]KasbonSaldo, error) {
	rows, err := t.tx.QueryContext(ctx, `
		SELECT k.id_kasbon, k.tanggal, k.jumlah,
		       COALESCE((
			SELECT SUM(pk.jumlah)
			FROM potongan_kasbon pk
			JOIN gaji g ON g.id_gaji = pk.id_gaji
			WHERE pk.id_kasbon = k.id_kasbon
			  AND g.status_bayar = 'dibayar'
		       ), '0.00') AS sudah_dibayar
		FROM kasbon k
		WHERE k.id_pekerja = ? AND k.tanggal <= ?
		ORDER BY k.tanggal ASC, k.id_kasbon ASC
		FOR UPDATE`,
		idPekerja,
		tanggalAkhir.Format("2006-01-02"),
	)
	if err != nil {
		return nil, fmt.Errorf("ambil kasbon bersaldo: %w", err)
	}
	defer rows.Close()

	var hasil []KasbonSaldo
	for rows.Next() {
		var item KasbonSaldo
		var jumlahStr, dibayarStr string
		if err := rows.Scan(&item.ID, &item.Tanggal, &jumlahStr, &dibayarStr); err != nil {
			return nil, fmt.Errorf("scan kasbon bersaldo: %w", err)
		}
		jumlahSen, err := parseDesimalSen(jumlahStr)
		if err != nil {
			return nil, err
		}
		dibayarSen, err := parseDesimalSen(dibayarStr)
		if err != nil {
			return nil, err
		}
		item.SaldoSen = jumlahSen - dibayarSen
		if item.SaldoSen > 0 {
			hasil = append(hasil, item)
		}
	}
	return hasil, rows.Err()
}

func (t *GajiTx) HapusPotonganKasbonDraf(ctx context.Context, idGaji int) error {
	_, err := t.tx.ExecContext(ctx, `
		DELETE pk
		FROM potongan_kasbon pk
		JOIN gaji g ON g.id_gaji = pk.id_gaji
		WHERE g.id_gaji = ? AND g.status_bayar = 'belum'`, idGaji)
	if err != nil {
		return fmt.Errorf("hapus potongan kasbon draf: %w", err)
	}
	return nil
}

func (t *GajiTx) TambahPotonganKasbon(ctx context.Context, idGaji, idKasbon int, jumlah string) error {
	_, err := t.tx.ExecContext(ctx, `
		INSERT INTO potongan_kasbon (id_gaji, id_kasbon, jumlah)
		VALUES (?, ?, ?)`, idGaji, idKasbon, jumlah)
	if err != nil {
		return fmt.Errorf("tambah potongan kasbon: %w", err)
	}
	return nil
}

func (t *GajiTx) TerapkanKoreksi(ctx context.Context, idGaji int, ids []int) error {
	for _, idKoreksi := range ids {
		res, err := t.tx.ExecContext(ctx, `
			UPDATE koreksi_gaji
			SET id_gaji_tujuan = ?, status = 'diterapkan'
			WHERE id_koreksi = ? AND status = 'disetujui' AND id_gaji_tujuan IS NULL`,
			idGaji, idKoreksi,
		)
		if err != nil {
			return fmt.Errorf("terapkan koreksi %d: %w", idKoreksi, err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("periksa penerapan koreksi %d: %w", idKoreksi, err)
		}
		if n != 1 {
			return &AturanError{Pesan: "Status koreksi berubah; hitung ulang draf sebelum membayar"}
		}
	}
	return nil
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
		AND (
			(
				p.tanggal_masuk <= pg.tanggal_akhir
				AND (p.tanggal_keluar IS NULL OR p.tanggal_keluar >= pg.tanggal_awal)
			)
			OR EXISTS (
				SELECT 1
				FROM koreksi_gaji kg
				JOIN gaji ga ON ga.id_gaji = kg.id_gaji_asal
				JOIN periode_gaji pa ON pa.id_periode = ga.id_periode
				WHERE ga.id_pekerja = p.id_pekerja
				  AND kg.status = 'disetujui'
				  AND kg.id_gaji_tujuan IS NULL
				  AND pa.jenis_periode = pg.jenis_periode
				  AND pa.tanggal_awal < pg.tanggal_awal
			)
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