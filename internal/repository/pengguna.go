package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"

	"github.com/anggyputra/go-class/internal/model"
)

type PenggunaRepo struct {
	DB *sql.DB
}

func (r *PenggunaRepo) TambahPengguna(ctx context.Context, u model.Pengguna) (int64, error) {
	aktif := 0
	if u.Aktif {
		aktif = 1
	}

	res, err := r.DB.ExecContext(ctx,
		`INSERT INTO pengguna (username, password_hash, nama_lengkap, peran, aktif)
		 VALUES (?, ?, ?, ?, ?)`,
		u.Username, u.PasswordHash, u.NamaLengkap, string(u.Peran), aktif,
	)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return 0, model.ErrUsernameSudahDipakai
		}
		return 0, fmt.Errorf("tambah pengguna: %w", err)
	}
	return res.LastInsertId()
}

func (r *PenggunaRepo) CariByUsername(ctx context.Context, username string) (model.Pengguna, error) {
	var u model.Pengguna
	var peran string
	var aktif int

	err := r.DB.QueryRowContext(ctx,
		`SELECT id_pengguna, username, password_hash, nama_lengkap, peran, aktif, created_at
		 FROM pengguna
		 WHERE username = ? AND aktif = 1`,
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.NamaLengkap, &peran, &aktif, &u.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return u, model.ErrNotFound
	}
	if err != nil {
		return u, fmt.Errorf("cari pengguna %s: %w", username, err)
	}

	u.Peran = model.PeranPengguna(peran)
	u.Aktif = aktif == 1
	return u, nil
}

func (r *PenggunaRepo) CariByID(ctx context.Context, id int64) (model.Pengguna, error) {
	var u model.Pengguna
	var peran string
	var aktif int

	err := r.DB.QueryRowContext(ctx,
		`SELECT id_pengguna, username, password_hash, nama_lengkap, peran, aktif, created_at
		 FROM pengguna
		 WHERE id_pengguna = ?`,
		id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.NamaLengkap, &peran, &aktif, &u.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return u, model.ErrNotFound
	}
	if err != nil {
		return u, fmt.Errorf("cari pengguna id %d: %w", id, err)
	}

	u.Peran = model.PeranPengguna(peran)
	u.Aktif = aktif == 1
	return u, nil
}

func (r *PenggunaRepo) SimpanSesi(ctx context.Context, s model.Sesi) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO sesi (token, id_pengguna, expired_at) VALUES (?, ?, ?)`,
		s.Token, s.IDPengguna, s.ExpiredAt,
	)
	if err != nil {
		return fmt.Errorf("simpan sesi: %w", err)
	}
	return nil
}

func (r *PenggunaRepo) CariSesiValid(ctx context.Context, token string) (model.Sesi, error) {
	var s model.Sesi
	var peran string
	var aktif int

	err := r.DB.QueryRowContext(ctx,
		`SELECT s.token, s.id_pengguna, s.expired_at, s.created_at,
		        p.id_pengguna, p.username, p.nama_lengkap, p.peran, p.aktif, p.created_at
		 FROM sesi s
		 JOIN pengguna p ON p.id_pengguna = s.id_pengguna
		 WHERE s.token = ? AND s.expired_at > NOW() AND p.aktif = 1`,
		token,
	).Scan(
		&s.Token, &s.IDPengguna, &s.ExpiredAt, &s.CreatedAt,
		&s.Pengguna.ID, &s.Pengguna.Username, &s.Pengguna.NamaLengkap, &peran, &aktif, &s.Pengguna.CreatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return s, model.ErrSesiTidakValid
	}
	if err != nil {
		return s, fmt.Errorf("cari sesi valid: %w", err)
	}

	s.Pengguna.Peran = model.PeranPengguna(peran)
	s.Pengguna.Aktif = aktif == 1
	return s, nil
}

func (r *PenggunaRepo) HapusSesi(ctx context.Context, token string) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM sesi WHERE token = ?`, token)
	if err != nil {
		return fmt.Errorf("hapus sesi: %w", err)
	}
	return nil
}

func (r *PenggunaRepo) HapusSesiKedaluwarsa(ctx context.Context) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM sesi WHERE expired_at <= NOW()`)
	return err
}

func (r *PenggunaRepo) HitungPengguna(ctx context.Context) (int, error) {
	var total int
	err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM pengguna`).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("hitung pengguna: %w", err)
	}
	return total, nil
}

// AmbilRingkasanDashboard mengambil ringkasan metrik utama untuk halaman Beranda.
func (r *PenggunaRepo) AmbilRingkasanDashboard(ctx context.Context) model.RingkasanDashboard {
	var d model.RingkasanDashboard

	_ = r.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM pekerja WHERE tanggal_keluar IS NULL`,
	).Scan(&d.TotalPekerjaAktif)

	_ = r.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM pekerja WHERE tanggal_keluar IS NULL AND jenis_pekerja = 'harian'`,
	).Scan(&d.PekerjaHarianAktif)

	_ = r.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM pekerja WHERE tanggal_keluar IS NULL AND jenis_pekerja = 'bulanan'`,
	).Scan(&d.PekerjaBulananAktif)

	_ = r.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM periode_gaji WHERE status = 'terbuka'`,
	).Scan(&d.PeriodeTerbuka)

	_ = r.DB.QueryRowContext(ctx,
		`SELECT COUNT(*)
		 FROM jadwal_shift js
		 LEFT JOIN absensi a ON a.id_jadwal = js.id_jadwal
		 WHERE a.id_absensi IS NULL OR a.status_kehadiran = 'perlu_diperiksa'`,
	).Scan(&d.AbsensiPerluPeriksa)

	_ = r.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM kasbon WHERE status = 'aktif' AND saldo_sisa > 0`,
	).Scan(&d.KasbonAktif)

	return d
}

