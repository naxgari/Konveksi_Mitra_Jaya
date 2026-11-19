package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/anggyputra/go-class/internal/model"
	"github.com/anggyputra/go-class/internal/repository"
)

func TestTutupPeriodeMySQL(t *testing.T) {
	db := bukaDBDrafUji(t)
	ctx := context.Background()

	// Buat akun Pemilik khusus pengujian.
	hasil, err := db.Exec(`
		INSERT INTO pengguna
			(username, password_hash, nama_lengkap, peran, aktif)
		VALUES (?, 'hash-khusus-pengujian', 'Pemilik Uji', 'pemilik', 1)`,
		fmt.Sprintf("pemilik-tutup-%d", time.Now().UnixNano()))
	if err != nil {
		t.Fatal(err)
	}

	idPemilik, err := hasil.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, err := db.Exec(
			"DELETE FROM pengguna WHERE id_pengguna = ?", idPemilik)
		if err != nil {
			t.Errorf("bersihkan Pemilik uji: %v", err)
		}
	})

	pemilik := model.Pengguna{
		ID:    idPemilik,
		Peran: model.PeranPemilik,
		Aktif: true,
	}

	idPekerja, idPeriode, idJadwal := siapkanDataDrafUji(t, db)

	// Siapkan periode bulanan untuk memastikan tidak ikut terkunci.
	idBulanan, periodeBulanan, jadwalBulanan := siapkanDataDrafUji(t, db)

	_, err = db.Exec(`
		UPDATE pekerja SET jenis_pekerja = 'bulanan'
		WHERE id_pekerja = ?`, idBulanan)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		UPDATE periode_gaji
		SET jenis_periode = 'bulanan',
		    tanggal_awal = '2093-01-01',
		    tanggal_akhir = '2093-01-31',
		    tanggal_bayar_rencana = '2093-01-31'
		WHERE id_periode = ?`, periodeBulanan)
	if err != nil {
		t.Fatal(err)
	}

	gaji := &GajiService{Repo: &repository.GajiRepo{DB: db}}
	periode := &PeriodeGajiService{
		Repo: &repository.PeriodeGajiRepo{DB: db},
	}
	absensi := &AbsensiService{
		Repo: &repository.AbsensiRepo{DB: db},
	}

	tolak := func(err error) {
		t.Helper()
		var validasi *ValidasiError
		if !errors.As(err, &validasi) {
			t.Fatalf("ingin error validasi, mendapat: %v", err)
		}
	}

	if _, err := gaji.HitungDraf(ctx, idPekerja, idPeriode, 0, 0); err != nil {
		t.Fatal(err)
	}

	// Admin tidak boleh menutup periode.
	admin := pemilik
	admin.Peran = model.PeranAdmin
	tolak(periode.Tutup(ctx, admin, idPeriode))

	// Pemilik juga ditolak jika masih ada gaji belum dibayar.
	tolak(periode.Tutup(ctx, pemilik, idPeriode))

	if err := gaji.Bayar(
		ctx, model.PeranAdmin, idPekerja, idPeriode,
	); err != nil {
		t.Fatal(err)
	}

	// Setelah dibayar, penutupan berhasil.
	if err := periode.Tutup(ctx, pemilik, idPeriode); err != nil {
		t.Fatalf("tutup periode: %v", err)
	}

	var status string
	var ditutupOleh int64
	var ditutupPada time.Time

	err = db.QueryRow(`
		SELECT status, ditutup_oleh, ditutup_pada
		FROM periode_gaji WHERE id_periode = ?`, idPeriode,
	).Scan(&status, &ditutupOleh, &ditutupPada)
	if err != nil {
		t.Fatal(err)
	}
	if status != "tertutup" ||
		ditutupOleh != idPemilik ||
		ditutupPada.IsZero() {
		t.Fatal("status, penutup, atau waktu penutupan tidak sesuai")
	}

	// Penutupan kedua harus ditolak.
	tolak(periode.Tutup(ctx, pemilik, idPeriode))

	var waktuSetelah time.Time
	err = db.QueryRow(`
		SELECT ditutup_pada FROM periode_gaji
		WHERE id_periode = ?`, idPeriode,
	).Scan(&waktuSetelah)
	if err != nil {
		t.Fatal(err)
	}
	if !waktuSetelah.Equal(ditutupPada) {
		t.Fatal("waktu penutupan pertama berubah")
	}

	// Absensi harian pada periode tertutup tidak dapat diubah.
	err = absensi.AbsensiSimpan(ctx, model.Absensi{
		IDJadwal: idJadwal,
		Status:   "tidak_hadir",
	})
	var terkunci *ValidasiError
	if !errors.As(err, &terkunci) ||
		terkunci.Pesan != "Periode gaji sudah ditutup" {
		t.Fatalf("ingin penolakan karena periode tertutup, mendapat: %v", err)
	}

	// Absensi bulanan tetap dapat diubah karena periodenya terbuka.
	err = absensi.AbsensiSimpan(ctx, model.Absensi{
		IDJadwal: jadwalBulanan,
		Status:   "tidak_hadir",
	})
	if err != nil {
		t.Fatalf("absensi bulanan ikut terkunci: %v", err)
	}

	_, err = gaji.HitungDraf(ctx, idPekerja, idPeriode, 0, 0)
	tolak(err)
}

