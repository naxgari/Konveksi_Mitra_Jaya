package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/anggyputra/go-class/internal/model"
	"github.com/go-sql-driver/mysql"
)

type AbsensiRepo struct{ DB *sql.DB }
type AturanError struct{ Pesan string }

func (e *AturanError) Error() string { return e.Pesan }
func aturan(s string) error          { return &AturanError{s} }
func dbErr(err error) error {
	var e *mysql.MySQLError
	if errors.As(err, &e) && e.Number == 1062 {
		return aturan("Data sudah ada; periksa nama shift atau jadwal pekerja pada tanggal tersebut")
	}
	return err
}
func (r *AbsensiRepo) ShiftDaftar(ctx context.Context) ([]model.Shift, error) {
	rows, e := r.DB.QueryContext(ctx, `SELECT id_shift,nama_shift,TIME_FORMAT(jam_mulai,'%H:%i'),TIME_FORMAT(jam_selesai,'%H:%i') FROM shift ORDER BY jam_mulai`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []model.Shift
	for rows.Next() {
		var v model.Shift
		if e = rows.Scan(&v.ID, &v.Nama, &v.JamMulai, &v.JamSelesai); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *AbsensiRepo) ShiftDetail(ctx context.Context, id int) (model.Shift, error) {
	var v model.Shift
	e := r.DB.QueryRowContext(ctx, `SELECT id_shift,nama_shift,TIME_FORMAT(jam_mulai,'%H:%i'),TIME_FORMAT(jam_selesai,'%H:%i') FROM shift WHERE id_shift=?`, id).Scan(&v.ID, &v.Nama, &v.JamMulai, &v.JamSelesai)
	if errors.Is(e, sql.ErrNoRows) {
		e = model.ErrNotFound
	}
	return v, e
}
func (r *AbsensiRepo) ShiftBuat(ctx context.Context, v model.Shift) (int64, error) {
	res, e := r.DB.ExecContext(ctx, `INSERT INTO shift (nama_shift,jam_mulai,jam_selesai) VALUES (?,?,?)`, v.Nama, v.JamMulai, v.JamSelesai)
	if e != nil {
		return 0, dbErr(e)
	}
	return res.LastInsertId()
}
func (r *AbsensiRepo) PekerjaDaftar(ctx context.Context) ([]model.PekerjaJadwal, error) {
	rows, e := r.DB.QueryContext(ctx, `SELECT id_pekerja,nama,jenis_pekerja FROM pekerja ORDER BY nama`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []model.PekerjaJadwal
	for rows.Next() {
		var v model.PekerjaJadwal
		if e = rows.Scan(&v.ID, &v.Nama, &v.Jenis); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

const jadwalSelect = `SELECT j.id_jadwal,j.id_pekerja,j.id_shift,p.nama,p.jenis_pekerja,s.nama_shift,TIME_FORMAT(s.jam_mulai,'%H:%i'),TIME_FORMAT(s.jam_selesai,'%H:%i'),j.tanggal,a.id_absensi,a.status_kehadiran,a.jam_masuk,a.jam_keluar,a.menit_terlambat,a.menit_lembur,a.keterangan FROM jadwal_shift j JOIN pekerja p ON p.id_pekerja=j.id_pekerja JOIN shift s ON s.id_shift=j.id_shift LEFT JOIN absensi a ON a.id_jadwal=j.id_jadwal `

type scanner interface{ Scan(...any) error }

func scanJadwal(row scanner) (model.JadwalShift, error) {
	var j model.JadwalShift
	var id, late, extra sql.NullInt64
	var status, ket sql.NullString
	var masuk, keluar sql.NullTime
	e := row.Scan(&j.ID, &j.IDPekerja, &j.IDShift, &j.NamaPekerja, &j.JenisPekerja, &j.NamaShift, &j.JamMulai, &j.JamSelesai, &j.Tanggal, &id, &status, &masuk, &keluar, &late, &extra, &ket)
	if errors.Is(e, sql.ErrNoRows) {
		return j, model.ErrNotFound
	}
	if e != nil {
		return j, e
	}
	if id.Valid {
		a := &model.Absensi{ID: int(id.Int64), IDJadwal: j.ID, Status: status.String, MenitTerlambat: int(late.Int64), MenitLembur: int(extra.Int64), Keterangan: ket.String}
		if masuk.Valid {
			a.JamMasuk = &masuk.Time
		}
		if keluar.Valid {
			a.JamKeluar = &keluar.Time
		}
		j.Absensi = a
	}
	return j, nil
}
func (r *AbsensiRepo) JadwalDaftar(ctx context.Context) ([]model.JadwalShift, error) {
	rows, e := r.DB.QueryContext(ctx, jadwalSelect+`ORDER BY j.tanggal DESC,p.nama`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []model.JadwalShift
	for rows.Next() {
		v, e := scanJadwal(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *AbsensiRepo) JadwalDetail(ctx context.Context, id int) (model.JadwalShift, error) {
	return scanJadwal(r.DB.QueryRowContext(ctx, jadwalSelect+`WHERE j.id_jadwal=?`, id))
}

// Transaksi khusus modul menjaga pembacaan aturan dan perubahan dalam transaksi yang sama.
type AbsensiTx struct{ tx *sql.Tx }

func (r *AbsensiRepo) Transaksi(ctx context.Context, fn func(*AbsensiTx) error) error {
	tx, e := r.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if e = fn(&AbsensiTx{tx}); e != nil {
		return e
	}
	return tx.Commit()
}
func (t *AbsensiTx) Shift(ctx context.Context, id int) (model.Shift, error) {
	var v model.Shift
	e := t.tx.QueryRowContext(ctx, `SELECT id_shift,nama_shift,TIME_FORMAT(jam_mulai,'%H:%i'),TIME_FORMAT(jam_selesai,'%H:%i') FROM shift WHERE id_shift=? FOR UPDATE`, id).Scan(&v.ID, &v.Nama, &v.JamMulai, &v.JamSelesai)
	if errors.Is(e, sql.ErrNoRows) {
		e = model.ErrNotFound
	}
	return v, e
}
func (t *AbsensiTx) JumlahJadwalShift(ctx context.Context, id int) (int, error) {
	var n int
	e := t.tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM jadwal_shift WHERE id_shift=?`, id).Scan(&n)
	return n, e
}
func (t *AbsensiTx) UbahShift(ctx context.Context, v model.Shift) error {
	_, e := t.tx.ExecContext(ctx, `UPDATE shift SET nama_shift=?,jam_mulai=?,jam_selesai=? WHERE id_shift=?`, v.Nama, v.JamMulai, v.JamSelesai, v.ID)
	return dbErr(e)
}
func (t *AbsensiTx) HapusShift(ctx context.Context, id int) error {
	_, e := t.tx.ExecContext(ctx, `DELETE FROM shift WHERE id_shift=?`, id)
	return e
}

type DataPekerja struct {
	Jenis  string
	Masuk  time.Time
	Keluar sql.NullTime
}

func (t *AbsensiTx) Pekerja(ctx context.Context, id int) (DataPekerja, error) {
	var p DataPekerja
	e := t.tx.QueryRowContext(ctx, `SELECT jenis_pekerja,tanggal_masuk,tanggal_keluar FROM pekerja WHERE id_pekerja=? FOR UPDATE`, id).Scan(&p.Jenis, &p.Masuk, &p.Keluar)
	if errors.Is(e, sql.ErrNoRows) {
		e = model.ErrNotFound
	}
	return p, e
}

type DataPeriode struct {
	ID          int
	Status      string
	StatusBayar string
}

func (t *AbsensiTx) Periode(ctx context.Context, id int, jenis, tgl string) ([]DataPeriode, error) {
	rows, e := t.tx.QueryContext(ctx, `SELECT id_periode,status FROM periode_gaji WHERE jenis_periode=? AND ? BETWEEN tanggal_awal AND tanggal_akhir FOR UPDATE`, jenis, tgl)
	if e != nil {
		return nil, e
	}
	var out []DataPeriode
	for rows.Next() {
		var p DataPeriode
		if e = rows.Scan(&p.ID, &p.Status); e != nil {
			rows.Close()
			return nil, e
		}
		out = append(out, p)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	for i := range out {
		e = t.tx.QueryRowContext(ctx, `SELECT status_bayar FROM gaji WHERE id_pekerja=? AND id_periode=? FOR UPDATE`, id, out[i].ID).Scan(&out[i].StatusBayar)
		if e != nil && !errors.Is(e, sql.ErrNoRows) {
			return nil, e
		}
	}
	return out, nil
}
func (t *AbsensiTx) Jadwal(ctx context.Context, id int) (model.JadwalShift, error) {
	return scanJadwal(t.tx.QueryRowContext(ctx, jadwalSelect+`WHERE j.id_jadwal=? FOR UPDATE`, id))
}
func (t *AbsensiTx) TambahJadwal(ctx context.Context, v model.JadwalShift) (int64, error) {
	res, e := t.tx.ExecContext(ctx, `INSERT INTO jadwal_shift(id_pekerja,id_shift,tanggal) VALUES(?,?,?)`, v.IDPekerja, v.IDShift, v.Tanggal.Format("2006-01-02"))
	if e != nil {
		return 0, dbErr(e)
	}
	return res.LastInsertId()
}
func (t *AbsensiTx) UbahJadwal(ctx context.Context, v model.JadwalShift) error {
	_, e := t.tx.ExecContext(ctx, `UPDATE jadwal_shift SET id_pekerja=?,id_shift=?,tanggal=? WHERE id_jadwal=?`, v.IDPekerja, v.IDShift, v.Tanggal.Format("2006-01-02"), v.ID)
	return dbErr(e)
}
func (t *AbsensiTx) HapusJadwal(ctx context.Context, id int) error {
	_, e := t.tx.ExecContext(ctx, `DELETE FROM jadwal_shift WHERE id_jadwal=?`, id)
	return e
}
func (t *AbsensiTx) SimpanAbsensi(ctx context.Context, a model.Absensi) error {
	_, e := t.tx.ExecContext(ctx, `INSERT INTO absensi(id_jadwal,status_kehadiran,jam_masuk,jam_keluar,menit_terlambat,menit_lembur,keterangan) VALUES(?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE status_kehadiran=VALUES(status_kehadiran),jam_masuk=VALUES(jam_masuk),jam_keluar=VALUES(jam_keluar),menit_terlambat=VALUES(menit_terlambat),menit_lembur=VALUES(menit_lembur),keterangan=VALUES(keterangan)`, a.IDJadwal, a.Status, a.JamMasuk, a.JamKeluar, a.MenitTerlambat, a.MenitLembur, a.Keterangan)
	if e != nil {
		return fmt.Errorf("simpan absensi: %w", e)
	}
	return nil
}
