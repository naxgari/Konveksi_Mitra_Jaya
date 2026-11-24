package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/anggyputra/go-class/internal/model"
)

// KasbonRepo mengelola tabel KASBON, POTONGAN_KASBON, KOREKSI_GAJI serta
// bacaan GAJI/PERIODE_GAJI/DETAIL_GAJI yang dibutuhkan modul Kasbon, Koreksi & Laporan.
type KasbonRepo struct{ DB *sql.DB }

// =========================================================================
// KASBON — daftar, detail, buat, hapus, riwayat potongan
// =========================================================================

const kasbonSelect = `SELECT k.id_kasbon,k.id_pekerja,p.nama,p.jenis_pekerja,k.tanggal,k.jumlah,COALESCE(k.keterangan,''),
COALESCE((SELECT SUM(pk.jumlah) FROM potongan_kasbon pk JOIN gaji g ON g.id_gaji=pk.id_gaji WHERE pk.id_kasbon=k.id_kasbon AND g.status_bayar='dibayar'),0) AS sudah_dipotong,
k.jumlah-COALESCE((SELECT SUM(pk.jumlah) FROM potongan_kasbon pk JOIN gaji g ON g.id_gaji=pk.id_gaji WHERE pk.id_kasbon=k.id_kasbon AND g.status_bayar='dibayar'),0) AS saldo
FROM kasbon k JOIN pekerja p ON p.id_pekerja=k.id_pekerja `

func scanKasbon(row scanner) (model.Kasbon, error) {
	var k model.Kasbon
	e := row.Scan(&k.ID, &k.IDPekerja, &k.NamaPekerja, &k.JenisPekerja, &k.Tanggal, &k.Jumlah, &k.Keterangan, &k.SudahDipotong, &k.Saldo)
	if errors.Is(e, sql.ErrNoRows) {
		return k, model.ErrNotFound
	}
	return k, e
}

// KasbonDaftar mengembalikan seluruh kasbon beserta saldo setelah potongan gaji dibayar.
func (r *KasbonRepo) KasbonDaftar(ctx context.Context) ([]model.Kasbon, error) {
	rows, e := r.DB.QueryContext(ctx, kasbonSelect+`ORDER BY k.tanggal DESC,k.id_kasbon DESC`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []model.Kasbon
	for rows.Next() {
		k, e := scanKasbon(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// KasbonDetail mengembalikan satu kasbon beserta saldonya.
func (r *KasbonRepo) KasbonDetail(ctx context.Context, id int) (model.Kasbon, error) {
	return scanKasbon(r.DB.QueryRowContext(ctx, kasbonSelect+`WHERE k.id_kasbon=?`, id))
}

// PekerjaAktif mengembalikan pekerja yang masih dalam masa aktif (untuk dropdown kasbon).
func (r *KasbonRepo) PekerjaAktif(ctx context.Context) ([]model.PekerjaJadwal, error) {
	rows, e := r.DB.QueryContext(ctx, `SELECT id_pekerja,nama,jenis_pekerja FROM pekerja WHERE tanggal_keluar IS NULL OR tanggal_keluar>=CURDATE() ORDER BY nama`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []model.PekerjaJadwal
	for rows.Next() {
		var p model.PekerjaJadwal
		if e = rows.Scan(&p.ID, &p.Nama, &p.Jenis); e != nil {
			return nil, e
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// PekerjaAdaAktif melaporkan apakah pekerja ada & masih aktif (untuk validasi pengajuan kasbon).
func (r *KasbonRepo) PekerjaAdaAktif(ctx context.Context, id int) (bool, error) {
	var keluar sql.NullTime
	e := r.DB.QueryRowContext(ctx, `SELECT tanggal_keluar FROM pekerja WHERE id_pekerja=?`, id).Scan(&keluar)
	if errors.Is(e, sql.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	if keluar.Valid && keluar.Time.Before(time.Now()) {
		return false, nil
	}
	return true, nil
}

// KasbonBuat mencatat pengajuan/pengambilan kasbon baru.
func (r *KasbonRepo) KasbonBuat(ctx context.Context, k model.Kasbon) (int64, error) {
	res, e := r.DB.ExecContext(ctx, `INSERT INTO kasbon(id_pekerja,tanggal,jumlah,keterangan) VALUES(?,?,?,?)`,
		k.IDPekerja, k.Tanggal.Format("2006-01-02"), k.Jumlah, k.Keterangan)
	if e != nil {
		return 0, dbErr(e)
	}
	return res.LastInsertId()
}

// JumlahPotonganKasbon mengembalikan total potongan yang sudah tertambat pada suatu kasbon.
func (r *KasbonRepo) JumlahPotonganKasbon(ctx context.Context, idKasbon int) (float64, error) {
	var n float64
	e := r.DB.QueryRowContext(ctx, `SELECT COALESCE(SUM(jumlah),0) FROM potongan_kasbon WHERE id_kasbon=?`, idKasbon).Scan(&n)
	return n, e
}

// KasbonHapus menghapus kasbon. Ditolak MySQL (FK RESTRICT) bila sudah dipotong.
func (r *KasbonRepo) KasbonHapus(ctx context.Context, id int) error {
	_, e := r.DB.ExecContext(ctx, `DELETE FROM kasbon WHERE id_kasbon=?`, id)
	if e != nil {
		return dbErr(e)
	}
	return nil
}

// PotonganByKasbon mengembalikan riwayat pemotongan sebuah kasbon (untuk halaman detail).
func (r *KasbonRepo) PotonganByKasbon(ctx context.Context, idKasbon int) ([]model.PotonganKasbon, error) {
	rows, e := r.DB.QueryContext(ctx, `SELECT pk.id_potongan,pk.id_gaji,pk.id_kasbon,pk.jumlah,
CONCAT(DATE_FORMAT(pg.tanggal_awal,'%d %b %Y'),' · ',pg.jenis_periode),g.status_bayar
FROM potongan_kasbon pk JOIN gaji g ON g.id_gaji=pk.id_gaji JOIN periode_gaji pg ON pg.id_periode=g.id_periode
WHERE pk.id_kasbon=? ORDER BY pg.tanggal_awal DESC`, idKasbon)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []model.PotonganKasbon
	for rows.Next() {
		var p model.PotonganKasbon
		if e = rows.Scan(&p.ID, &p.IDGaji, &p.IDKasbon, &p.Jumlah, &p.PeriodeLabel, &p.StatusBayar); e != nil {
			return nil, e
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// =========================================================================
// KOREKSI GAJI — daftar, pilihan gaji asal, buat, detail, setujui, hapus
// =========================================================================

const koreksiSelect = `SELECT k.id_koreksi,k.id_gaji_asal,COALESCE(k.id_gaji_tujuan,0),p.nama,
DATE_FORMAT(pa.tanggal_awal,'%d %b %Y'),COALESCE(DATE_FORMAT(pt.tanggal_awal,'%d %b %Y'),'—'),
k.dicatat_oleh,COALESCE(u1.nama_lengkap,''),COALESCE(k.disetujui_oleh,0),COALESCE(u2.nama_lengkap,''),
k.tanggal,k.nilai_selisih,k.alasan,k.status,k.disetujui_pada
FROM koreksi_gaji k
JOIN gaji ga ON ga.id_gaji=k.id_gaji_asal JOIN pekerja p ON p.id_pekerja=ga.id_pekerja
JOIN periode_gaji pa ON pa.id_periode=ga.id_periode
LEFT JOIN gaji gt ON gt.id_gaji=k.id_gaji_tujuan LEFT JOIN periode_gaji pt ON pt.id_periode=gt.id_periode
LEFT JOIN pengguna u1 ON u1.id_pengguna=k.dicatat_oleh LEFT JOIN pengguna u2 ON u2.id_pengguna=k.disetujui_oleh `

func scanKoreksi(row scanner) (model.KoreksiGaji, error) {
	var k model.KoreksiGaji
	var disetujui sql.NullTime
	e := row.Scan(&k.ID, &k.IDGajiAsal, &k.IDGajiTujuan, &k.NamaPekerja, &k.PeriodeAsal, &k.PeriodeTujuan,
		&k.DicatatOleh, &k.NamaPencatat, &k.DisetujuiOleh, &k.NamaPenyetuju,
		&k.Tanggal, &k.NilaiSelisih, &k.Alasan, &k.Status, &disetujui)
	if errors.Is(e, sql.ErrNoRows) {
		return k, model.ErrNotFound
	}
	if e != nil {
		return k, e
	}
	if disetujui.Valid {
		k.DisetujuiPada = disetujui.Time
	}
	return k, nil
}

// KoreksiDaftar mengembalikan seluruh koreksi gaji beserta label periode asal/tujuan.
func (r *KasbonRepo) KoreksiDaftar(ctx context.Context) ([]model.KoreksiGaji, error) {
	rows, e := r.DB.QueryContext(ctx, koreksiSelect+`ORDER BY k.created_at DESC`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []model.KoreksiGaji
	for rows.Next() {
		k, e := scanKoreksi(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// KoreksiDetail mengembalikan satu koreksi gaji.
func (r *KasbonRepo) KoreksiDetail(ctx context.Context, id int) (model.KoreksiGaji, error) {
	return scanKoreksi(r.DB.QueryRowContext(ctx, koreksiSelect+`WHERE k.id_koreksi=?`, id))
}

// PilihanGajiDibayar mengembalikan daftar gaji berstatus dibayar untuk dropdown gaji asal.
func (r *KasbonRepo) PilihanGajiDibayar(ctx context.Context) ([]model.PilihanGaji, error) {
	rows, e := r.DB.QueryContext(ctx, `SELECT g.id_gaji,CONCAT(p.nama,' · ',DATE_FORMAT(pg.tanggal_awal,'%d %b %Y'),'–',DATE_FORMAT(pg.tanggal_akhir,'%d %b %Y'),' · ',pg.jenis_periode)
FROM gaji g JOIN pekerja p ON p.id_pekerja=g.id_pekerja JOIN periode_gaji pg ON pg.id_periode=g.id_periode
WHERE g.status_bayar='dibayar' ORDER BY pg.tanggal_awal DESC,p.nama`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []model.PilihanGaji
	for rows.Next() {
		var v model.PilihanGaji
		if e = rows.Scan(&v.ID, &v.Label); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// GajiAsalDibayar memastikan koreksi hanya merujuk pada gaji yang telah dibayar.
func (r *KasbonRepo) GajiAsalDibayar(ctx context.Context, id int) (bool, error) {
	var ada bool
	e := r.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM gaji WHERE id_gaji=? AND status_bayar='dibayar')`, id).Scan(&ada)
	return ada, e
}

// KoreksiBuat mencatat pengajuan koreksi gaji (status diajukan).
func (r *KasbonRepo) KoreksiBuat(ctx context.Context, k model.KoreksiGaji) (int64, error) {
	res, e := r.DB.ExecContext(ctx, `INSERT INTO koreksi_gaji(id_gaji_asal,dicatat_oleh,tanggal,nilai_selisih,alasan,status) VALUES(?,?,?,?,?,'diajukan')`,
		k.IDGajiAsal, k.DicatatOleh, k.Tanggal.Format("2006-01-02"), k.NilaiSelisih, k.Alasan)
	if e != nil {
		return 0, dbErr(e)
	}
	return res.LastInsertId()
}

// KoreksiSetujui menandai koreksi disetujui pemilik. Hanya berlaku untuk status diajukan.
func (r *KasbonRepo) KoreksiSetujui(ctx context.Context, id, idPemilik int) error {
	res, e := r.DB.ExecContext(ctx, `UPDATE koreksi_gaji SET status='disetujui',disetujui_oleh=?,disetujui_pada=NOW() WHERE id_koreksi=? AND status='diajukan'`, idPemilik, id)
	if e != nil {
		return e
	}
	n, e := res.RowsAffected()
	if e != nil {
		return e
	}
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// KoreksiHapus hanya menghapus pengajuan yang belum disetujui.
func (r *KasbonRepo) KoreksiHapus(ctx context.Context, id int) error {
	res, e := r.DB.ExecContext(ctx, `DELETE FROM koreksi_gaji WHERE id_koreksi=? AND status='diajukan'`, id)
	if e != nil {
		return e
	}
	n, e := res.RowsAffected()
	if e != nil {
		return e
	}
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// =========================================================================
// LAPORAN — slip gaji individual & rekap biaya gaji/lembur per periode
// =========================================================================

// AmbilSlip mengumpulkan seluruh data untuk slip gaji individual.
func (r *KasbonRepo) AmbilSlip(ctx context.Context, idGaji int) (model.SlipGaji, error) {
	var s model.SlipGaji
	var dibayar sql.NullTime
	e := r.DB.QueryRowContext(ctx, `SELECT g.id_gaji,g.gaji_pokok,g.total_lembur,g.potongan_telat,g.total_potongan_kasbon,g.total_koreksi,g.gaji_bruto,g.gaji_bersih,g.status_bayar,g.dibayar_pada,
p.id_pekerja,p.nama,p.jenis_pekerja,p.jabatan,pg.id_periode,pg.jenis_periode,pg.tanggal_awal,pg.tanggal_akhir,pg.status
FROM gaji g JOIN pekerja p ON p.id_pekerja=g.id_pekerja JOIN periode_gaji pg ON pg.id_periode=g.id_periode WHERE g.id_gaji=?`, idGaji).Scan(
		&s.Gaji.ID, &s.Gaji.GajiPokok, &s.Gaji.TotalLembur, &s.Gaji.PotonganTelat, &s.Gaji.TotalPotonganKasbon, &s.Gaji.TotalKoreksi, &s.Gaji.GajiBruto, &s.Gaji.GajiBersih, &s.Gaji.StatusBayar, &dibayar,
		&s.Pekerja.ID, &s.Pekerja.Nama, &s.Pekerja.Jenis, &s.Pekerja.Jabatan, &s.Periode.ID, &s.Periode.JenisPeriode, &s.Periode.TanggalAwal, &s.Periode.TanggalAkhir, &s.Periode.Status)
	if errors.Is(e, sql.ErrNoRows) {
		return s, model.ErrNotFound
	}
	if e != nil {
		return s, e
	}
	if dibayar.Valid {
		s.Gaji.DibayarPada = dibayar.Time
	}
	s.BisaPotong = s.Gaji.StatusBayar == "belum" && s.Periode.Status == "terbuka"

	dRows, e := r.DB.QueryContext(ctx, `SELECT d.tanggal,d.nilai_pokok,d.menit_telat,d.jam_lembur,d.nilai_lembur,d.potongan_telat,
COALESCE(s.nama_shift,''),COALESCE(TIME_FORMAT(a.jam_masuk,'%H:%i'),'-'),COALESCE(TIME_FORMAT(a.jam_keluar,'%H:%i'),'-')
FROM detail_gaji d LEFT JOIN absensi a ON a.id_absensi=d.id_absensi LEFT JOIN jadwal_shift j ON j.id_jadwal=a.id_jadwal LEFT JOIN shift s ON s.id_shift=j.id_shift
WHERE d.id_gaji=? ORDER BY d.tanggal`, idGaji)
	if e != nil {
		return s, e
	}
	defer dRows.Close()
	for dRows.Next() {
		var b model.SlipBaris
		if e = dRows.Scan(&b.Tanggal, &b.NilaiPokok, &b.MenitTelat, &b.JamLembur, &b.NilaiLembur, &b.PotonganTelat, &b.NamaShift, &b.JamMasuk, &b.JamKeluar); e != nil {
			return s, e
		}
		s.Detail = append(s.Detail, b)
	}
	if e = dRows.Err(); e != nil {
		return s, e
	}

	pRows, e := r.DB.QueryContext(ctx, `SELECT k.tanggal,pk.jumlah,k.keterangan FROM potongan_kasbon pk JOIN kasbon k ON k.id_kasbon=pk.id_kasbon WHERE pk.id_gaji=? ORDER BY k.tanggal`, idGaji)
	if e != nil {
		return s, e
	}
	defer pRows.Close()
	for pRows.Next() {
		var p model.SlipPotongan
		if e = pRows.Scan(&p.TanggalKasbon, &p.Jumlah, &p.Keterangan); e != nil {
			return s, e
		}
		s.PotonganKasbon = append(s.PotonganKasbon, p)
	}
	if e = pRows.Err(); e != nil {
		return s, e
	}

	kRows, e := r.DB.QueryContext(ctx, `SELECT tanggal,nilai_selisih,alasan,status FROM koreksi_gaji WHERE id_gaji_tujuan=? ORDER BY tanggal`, idGaji)
	if e != nil {
		return s, e
	}
	defer kRows.Close()
	for kRows.Next() {
		var k model.SlipKoreksi
		if e = kRows.Scan(&k.Tanggal, &k.NilaiSelisih, &k.Alasan, &k.Status); e != nil {
			return s, e
		}
		s.Koreksi = append(s.Koreksi, k)
	}
	return s, kRows.Err()
}

// PilihanPeriode mengembalikan daftar periode gaji untuk filter laporan.
func (r *KasbonRepo) PilihanPeriode(ctx context.Context) ([]model.PilihanPeriode, error) {
	rows, e := r.DB.QueryContext(ctx, `SELECT id_periode,CONCAT(DATE_FORMAT(tanggal_awal,'%d %b %Y'),' – ',DATE_FORMAT(tanggal_akhir,'%d %b %Y'),' · ',jenis_periode) FROM periode_gaji ORDER BY tanggal_awal DESC`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []model.PilihanPeriode
	for rows.Next() {
		var v model.PilihanPeriode
		if e = rows.Scan(&v.ID, &v.Label); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// RekapDaftar mengembalikan baris gaji terfilter periode & jenis pekerja beserta totalnya.
func (r *KasbonRepo) RekapDaftar(ctx context.Context, idPeriode int, jenisPekerja string) ([]model.RekapBaris, model.RingkasanRekap, error) {
	q := `SELECT g.id_gaji,p.id_pekerja,p.nama,CONCAT(DATE_FORMAT(pg.tanggal_awal,'%d %b %Y'),'–',DATE_FORMAT(pg.tanggal_akhir,'%d %b %Y')),p.jenis_pekerja,g.gaji_pokok,g.total_lembur,g.potongan_telat,g.total_potongan_kasbon,g.total_koreksi,g.gaji_bruto,g.gaji_bersih,g.status_bayar
FROM gaji g JOIN pekerja p ON p.id_pekerja=g.id_pekerja JOIN periode_gaji pg ON pg.id_periode=g.id_periode WHERE 1=1`
	args := []any{}
	if idPeriode > 0 {
		q += ` AND g.id_periode=?`
		args = append(args, idPeriode)
	}
	if jenisPekerja == "harian" || jenisPekerja == "bulanan" {
		q += ` AND p.jenis_pekerja=?`
		args = append(args, jenisPekerja)
	}
	q += ` ORDER BY pg.tanggal_awal DESC,p.nama`
	rows, e := r.DB.QueryContext(ctx, q, args...)
	if e != nil {
		return nil, model.RingkasanRekap{}, e
	}
	defer rows.Close()
	var out []model.RekapBaris
	var ring model.RingkasanRekap
	for rows.Next() {
		var b model.RekapBaris
		if e = rows.Scan(&b.IDGaji, &b.IDPekerja, &b.NamaPekerja, &b.Periode, &b.JenisPekerja, &b.GajiPokok, &b.TotalLembur, &b.PotonganTelat, &b.PotonganKasbon, &b.TotalKoreksi, &b.GajiBruto, &b.GajiBersih, &b.StatusBayar); e != nil {
			return nil, ring, e
		}
		out = append(out, b)
		ring.TotalGajiPokok += b.GajiPokok
		ring.TotalLembur += b.TotalLembur
		ring.TotalPotonganTelat += b.PotonganTelat
		ring.TotalPotonganKasbon += b.PotonganKasbon
		ring.TotalKoreksi += b.TotalKoreksi
		ring.TotalGajiBruto += b.GajiBruto
		ring.TotalGajiBersih += b.GajiBersih
		ring.JumlahGaji++
		if b.StatusBayar == "dibayar" {
			ring.JumlahDibayar++
		}
	}
	return out, ring, rows.Err()
}

// =========================================================================
// TRANSAKSI — engine pemotongan kasbon (AB 08) & penautan koreksi (AB 11)
// =========================================================================

// KasbonTx membungkus sql.Tx untuk mutasi transaksional modul kasbon/koreksi.
type KasbonTx struct{ tx *sql.Tx }

// Transaksi menjalankan fn dalam satu transaksi database (rollback otomatis bila gagal).
func (r *KasbonRepo) Transaksi(ctx context.Context, fn func(*KasbonTx) error) error {
	tx, e := r.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if e = fn(&KasbonTx{tx}); e != nil {
		return e
	}
	return tx.Commit()
}

// DataGaji adalah ringkasan gaji + periode terkait yang dikunci saat engine berjalan.
type DataGaji struct {
	IDPekerja           int
	IDPeriode           int
	GajiBruto           float64
	PotonganTelat       float64
	TotalPotonganKasbon float64
	TotalKoreksi        float64
	StatusBayar         string
	TanggalAkhir        time.Time
	TanggalAwal         time.Time
	StatusPeriode       string
}

// Gaji mengunci baris gaji + periode terkait untuk engine (FOR UPDATE).
func (t *KasbonTx) Gaji(ctx context.Context, idGaji int) (DataGaji, error) {
	var d DataGaji
	e := t.tx.QueryRowContext(ctx, `SELECT g.id_pekerja,g.id_periode,g.gaji_bruto,g.potongan_telat,g.total_potongan_kasbon,g.total_koreksi,g.status_bayar,pg.tanggal_akhir,pg.tanggal_awal,pg.status
FROM gaji g JOIN periode_gaji pg ON pg.id_periode=g.id_periode WHERE g.id_gaji=? FOR UPDATE`, idGaji).
		Scan(&d.IDPekerja, &d.IDPeriode, &d.GajiBruto, &d.PotonganTelat, &d.TotalPotonganKasbon, &d.TotalKoreksi, &d.StatusBayar, &d.TanggalAkhir, &d.TanggalAwal, &d.StatusPeriode)
	if errors.Is(e, sql.ErrNoRows) {
		return d, model.ErrNotFound
	}
	return d, e
}

// HapusPotonganByGaji menghapus seluruh potongan kasbon pada gaji (idempoten untuk hitung ulang).
func (t *KasbonTx) HapusPotonganByGaji(ctx context.Context, idGaji int) error {
	_, e := t.tx.ExecContext(ctx, `DELETE FROM potongan_kasbon WHERE id_gaji=?`, idGaji)
	return e
}

// KasbonKandidat adalah kasbon bersaldo yang dapat dipotong pada periode tertentu.
type KasbonKandidat struct {
	ID    int
	Saldo float64
}

// KasbonUntukPotong mengunci & mengembalikan kasbon bersaldo (>0) milik pekerja
// yang tanggalnya tidak melewati akhir periode, urut tertua lebih dulu (FIFO).
func (t *KasbonTx) KasbonUntukPotong(ctx context.Context, idPekerja int, tanggalAkhir time.Time) ([]KasbonKandidat, error) {
	rows, e := t.tx.QueryContext(ctx, `SELECT k.id_kasbon,k.jumlah-COALESCE((SELECT SUM(pk.jumlah) FROM potongan_kasbon pk JOIN gaji g ON g.id_gaji=pk.id_gaji WHERE pk.id_kasbon=k.id_kasbon AND g.status_bayar='dibayar'),0) AS saldo
FROM kasbon k WHERE k.id_pekerja=? AND k.tanggal<=?
HAVING saldo>0 ORDER BY k.tanggal ASC,k.id_kasbon ASC FOR UPDATE`, idPekerja, tanggalAkhir.Format("2006-01-02"))
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []KasbonKandidat
	for rows.Next() {
		var c KasbonKandidat
		if e = rows.Scan(&c.ID, &c.Saldo); e != nil {
			return nil, e
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// TambahPotongan mencatat satu baris potongan kasbon pada gaji.
func (t *KasbonTx) TambahPotongan(ctx context.Context, idGaji, idKasbon int, jumlah float64) error {
	_, e := t.tx.ExecContext(ctx, `INSERT INTO potongan_kasbon(id_gaji,id_kasbon,jumlah) VALUES(?,?,?)`, idGaji, idKasbon, jumlah)
	return dbErr(e)
}

// TotalPotonganByGaji menghitung ulang total potongan kasbon pada gaji (SUM baris terbaru).
func (t *KasbonTx) TotalPotonganByGaji(ctx context.Context, idGaji int) (float64, error) {
	var n float64
	e := t.tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(jumlah),0) FROM potongan_kasbon WHERE id_gaji=?`, idGaji).Scan(&n)
	return n, e
}

// UpdateGajiPotongan memperbarui total_potongan_kasbon & gaji_bersih.
// gaji_bersih = gaji_bruto - potongan_telat - total_potongan_kasbon + total_koreksi.
func (t *KasbonTx) UpdateGajiPotongan(ctx context.Context, idGaji int, d DataGaji, totalPotongan, totalKoreksi float64) error {
	bersih := d.GajiBruto - d.PotonganTelat - totalPotongan + totalKoreksi
	_, e := t.tx.ExecContext(ctx, `UPDATE gaji SET total_potongan_kasbon=?,gaji_bersih=? WHERE id_gaji=?`, totalPotongan, bersih, idGaji)
	return e
}

// KoreksiDisetujuiUntuk mengunci koreksi disetujui yang ditujukan untuk periode gaji
// berikutnya yang tersedia bagi pekerja dan belum pernah diterapkan.
func (t *KasbonTx) KoreksiDisetujuiUntuk(ctx context.Context, idPekerja, idGajiTujuan int) ([]model.KoreksiGaji, error) {
	rows, e := t.tx.QueryContext(ctx, `SELECT k.id_koreksi,k.nilai_selisih,k.alasan,k.tanggal
FROM koreksi_gaji k JOIN gaji ga ON ga.id_gaji=k.id_gaji_asal JOIN periode_gaji pa ON pa.id_periode=ga.id_periode
JOIN gaji gt ON gt.id_gaji=? AND gt.id_pekerja=ga.id_pekerja JOIN periode_gaji pt ON pt.id_periode=gt.id_periode
WHERE ga.id_pekerja=? AND k.status='disetujui' AND k.id_gaji_tujuan IS NULL
AND pt.tanggal_awal>pa.tanggal_awal AND pt.jenis_periode=pa.jenis_periode
AND NOT EXISTS (
  SELECT 1 FROM gaji next_gaji JOIN periode_gaji next_period ON next_period.id_periode=next_gaji.id_periode
  WHERE next_gaji.id_pekerja=ga.id_pekerja AND next_period.jenis_periode=pa.jenis_periode
    AND next_period.tanggal_awal>pa.tanggal_awal AND next_period.tanggal_awal<pt.tanggal_awal
) ORDER BY pa.tanggal_awal,k.id_koreksi FOR UPDATE`,
		idGajiTujuan, idPekerja)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []model.KoreksiGaji
	for rows.Next() {
		var k model.KoreksiGaji
		if e = rows.Scan(&k.ID, &k.NilaiSelisih, &k.Alasan, &k.Tanggal); e != nil {
			return nil, e
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// TautkanKoreksi menautkan koreksi ke gaji tujuan & mengubah status menjadi diterapkan.
func (t *KasbonTx) TautkanKoreksi(ctx context.Context, idKoreksi, idGajiTujuan int) error {
	res, e := t.tx.ExecContext(ctx, `UPDATE koreksi_gaji SET id_gaji_tujuan=?,status='diterapkan' WHERE id_koreksi=? AND status='disetujui' AND id_gaji_tujuan IS NULL`, idGajiTujuan, idKoreksi)
	if e != nil {
		return e
	}
	n, e := res.RowsAffected()
	if e != nil {
		return e
	}
	if n != 1 {
		return model.ErrNotFound
	}
	return nil
}

// TotalKoreksiByGaji menghitung ulang total koreksi pada gaji (SUM nilai_selisih terbaru).
func (t *KasbonTx) TotalKoreksiByGaji(ctx context.Context, idGaji int) (float64, error) {
	var n float64
	e := t.tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(nilai_selisih),0) FROM koreksi_gaji WHERE id_gaji_tujuan=?`, idGaji).Scan(&n)
	return n, e
}

// UpdateGajiKoreksi memperbarui total_koreksi & gaji_bersih.
// gaji_bersih = gaji_bruto - potongan_telat - total_potongan_kasbon + total_koreksi.
func (t *KasbonTx) UpdateGajiKoreksi(ctx context.Context, idGaji int, d DataGaji, totalPotongan, totalKoreksi float64) error {
	bersih := d.GajiBruto - d.PotonganTelat - totalPotongan + totalKoreksi
	_, e := t.tx.ExecContext(ctx, `UPDATE gaji SET total_koreksi=?,gaji_bersih=? WHERE id_gaji=?`, totalKoreksi, bersih, idGaji)
	return e
}
