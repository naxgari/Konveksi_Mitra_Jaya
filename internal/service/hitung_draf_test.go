package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/anggyputra/go-class/internal/model"
	"github.com/anggyputra/go-class/internal/repository"
	"github.com/go-sql-driver/mysql"
)

func bukaDBDrafUji(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("DAFFA_TEST_DSN")
	if dsn == "" {
		t.Skip("DAFFA_TEST_DSN belum diisi")
	}

	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DBName != "konveksi_test" {
		t.Fatal("Tes ini harus memakai database konveksi_test")
	}
	cfg.ParseTime = true
	cfg.Loc = time.Local

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func siapkanDataDrafUji(
	t *testing.T,
	db *sql.DB,
) (idPekerja, idPeriode, idJadwal int) {
	t.Helper()

	var idShift int

	t.Cleanup(func() {
		hapus := func(query string, id int) {
			if id == 0 {
				return
			}
			if _, err := db.Exec(query, id); err != nil {
				t.Errorf("bersihkan data uji: %v", err)
			}
		}

		hapus("DELETE FROM gaji WHERE id_pekerja = ?", idPekerja)
		hapus(`DELETE a FROM absensi a
			JOIN jadwal_shift j ON j.id_jadwal = a.id_jadwal
			WHERE j.id_pekerja = ?`, idPekerja)
		hapus("DELETE FROM jadwal_shift WHERE id_pekerja = ?", idPekerja)
		hapus("DELETE FROM riwayat_upah WHERE id_pekerja = ?", idPekerja)
		hapus("DELETE FROM pekerja WHERE id_pekerja = ?", idPekerja)
		hapus("DELETE FROM periode_gaji WHERE id_periode = ?", idPeriode)
		hapus("DELETE FROM shift WHERE id_shift = ?", idShift)
	})

	tambah := func(query string, args ...any) int {
		t.Helper()

		hasil, err := db.Exec(query, args...)
		if err != nil {
			t.Fatal(err)
		}
		id, err := hasil.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		return int(id)
	}

	idPekerja = tambah(`
		INSERT INTO pekerja
			(nama, jenis_pekerja, jabatan, tanggal_masuk)
		VALUES ('Pekerja Uji Draf', 'harian', 'Penjahit', '2093-01-01')`)

	idPeriode = tambah(`
		INSERT INTO periode_gaji
			(jenis_periode, tanggal_awal, tanggal_akhir, tanggal_bayar_rencana)
		VALUES ('mingguan', '2093-01-05', '2093-01-10', '2093-01-10')`)

	idShift = tambah(`
		INSERT INTO shift (nama_shift, jam_mulai, jam_selesai)
		VALUES (?, '08:00:00', '16:00:00')`,
		fmt.Sprintf("draf-uji-%d", time.Now().UnixNano()))

	tambah(`
		INSERT INTO riwayat_upah (id_pekerja, berlaku_mulai, nominal)
		VALUES (?, '2093-01-01', '120000.00')`, idPekerja)

	idJadwal = tambah(`
		INSERT INTO jadwal_shift (id_pekerja, id_shift, tanggal)
		VALUES (?, ?, '2093-01-05')`, idPekerja, idShift)

	tambah(`
		INSERT INTO absensi
			(id_jadwal, status_kehadiran, jam_masuk, jam_keluar)
		VALUES (?, 'hadir', '2093-01-05 08:20:00', '2093-01-05 18:00:00')`,
		idJadwal)

	return
}

func TestHitungDrafMySQL(t *testing.T) {
	db := bukaDBDrafUji(t)
	idPekerja, idPeriode, idJadwal := siapkanDataDrafUji(t, db)

	ctx := context.Background()
	repo := &repository.GajiRepo{DB: db}
	s := &GajiService{Repo: repo}

	// 1. Buat draf: upah 120.000, lembur 2 jam, telat 20 menit.
	id, err := s.HitungDraf(ctx, idPekerja, idPeriode, 0, 0)
	if err != nil {
		t.Fatalf("buat draf: %v", err)
	}

	gaji, err := repo.CariByID(ctx, int(id))
	if err != nil {
		t.Fatal(err)
	}
	if gaji.GajiPokok != "120000.00" ||
		gaji.TotalLembur != "52500.00" ||
		gaji.PotonganTelat != "15000.00" ||
		gaji.GajiBruto != "172500.00" ||
		gaji.GajiBersih != "157500.00" ||
		gaji.StatusBayar != "belum" {
		t.Fatalf("hasil draf tidak sesuai: %+v", gaji)
	}

	detail, err := repo.DetailPerGaji(ctx, int(id))
	if err != nil {
		t.Fatal(err)
	}
	if len(detail) != 1 {
		t.Fatalf("ingin 1 detail, mendapat %d", len(detail))
	}
	if detail[0].MenitTelat != 20 ||
		detail[0].JamLembur != "2.000000" ||
		detail[0].NilaiLembur != "52500.00" {
		t.Fatalf("rincian tidak sesuai: %+v", detail[0])
	}

	// 2. Ubah jam keluar, lalu hitung ulang draf yang sama.
	_, err = db.Exec(`
		UPDATE absensi SET jam_keluar = '2093-01-05 16:00:00'
		WHERE id_jadwal = ?`, idJadwal)
	if err != nil {
		t.Fatal(err)
	}

	idLagi, err := s.HitungDraf(ctx, idPekerja, idPeriode, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if idLagi != id {
		t.Fatal("hitung ulang seharusnya memakai ID gaji yang sama")
	}

	gaji, err = repo.CariByID(ctx, int(id))
	if err != nil {
		t.Fatal(err)
	}
	if gaji.TotalLembur != "0.00" || gaji.GajiBersih != "105000.00" {
		t.Fatalf("hasil hitung ulang tidak sesuai: %+v", gaji)
	}

	detail, err = repo.DetailPerGaji(ctx, int(id))
	if err != nil {
		t.Fatal(err)
	}
	if len(detail) != 1 || detail[0].JamLembur != "0.000000" {
		t.Fatalf("detail hitung ulang tidak sesuai: %+v", detail)
	}

	// 3. Paksa kegagalan setelah header diubah dan detail diganti.
	// Dua detail yang sama akan melanggar aturan UNIQUE.
	idDetailLama := detail[0].ID
	ubah := gaji
	ubah.GajiPokok = "1.00"

	err = repo.Transaksi(ctx, func(tx *repository.GajiTx) error {
		_, err := simpanDraf(ctx, tx, ubah,
			[]model.DetailGaji{detail[0], detail[0]})
		return err
	})

	var duplikat *mysql.MySQLError
	if !errors.As(err, &duplikat) || duplikat.Number != 1062 {
		t.Fatalf("ingin error detail duplikat, mendapat: %v", err)
	}

	tersimpan, err := repo.CariByID(ctx, int(id))
	if err != nil {
		t.Fatal(err)
	}
	if tersimpan.GajiPokok != gaji.GajiPokok ||
		tersimpan.GajiBersih != gaji.GajiBersih {
		t.Fatalf("perubahan header tidak di-rollback: %+v", tersimpan)
	}

	detail, err = repo.DetailPerGaji(ctx, int(id))
	if err != nil {
		t.Fatal(err)
	}
	if len(detail) != 1 || detail[0].ID != idDetailLama {
		t.Fatalf("detail lama tidak dipulihkan oleh rollback: %+v", detail)
	}

	// 4. Absensi yang belum lengkap harus menolak perhitungan.
	_, err = db.Exec(`
		UPDATE absensi SET status_kehadiran = 'perlu_diperiksa'
		WHERE id_jadwal = ?`, idJadwal)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.HitungDraf(ctx, idPekerja, idPeriode, 0, 0)
	var validasi *ValidasiError
	if !errors.As(err, &validasi) {
		t.Fatalf("ingin error validasi absensi, mendapat: %v", err)
	}
}

func TestHitungDrafBulananMySQL(t *testing.T) {
	db := bukaDBDrafUji(t)
	idPekerja, idPeriode, idJadwal := siapkanDataDrafUji(t, db)

	ctx := context.Background()
	repo := &repository.GajiRepo{DB: db}
	s := &GajiService{Repo: repo}

	// Pekerja bulanan mulai aktif tanggal 16 Januari.
	_, err := db.Exec(`
		UPDATE pekerja
		SET jenis_pekerja = 'bulanan', tanggal_masuk = '2093-01-16'
		WHERE id_pekerja = ?`, idPekerja)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`
		UPDATE periode_gaji
		SET jenis_periode = 'bulanan',
		    tanggal_awal = '2093-01-01',
		    tanggal_akhir = '2093-01-31',
		    tanggal_bayar_rencana = '2093-01-31'
		WHERE id_periode = ?`, idPeriode)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`
		UPDATE riwayat_upah SET nominal = '3100000.00'
		WHERE id_pekerja = ?`, idPekerja)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`
		UPDATE jadwal_shift SET tanggal = '2093-01-16'
		WHERE id_jadwal = ?`, idJadwal)
	if err != nil {
		t.Fatal(err)
	}

	// Tetap telat 20 menit dan lembur 2 jam pada tanggal 16.
	_, err = db.Exec(`
		UPDATE absensi
		SET jam_masuk = '2093-01-16 08:20:00',
		    jam_keluar = '2093-01-16 18:00:00'
		WHERE id_jadwal = ?`, idJadwal)
	if err != nil {
		t.Fatal(err)
	}

	// 1. Prorata: 16 hari aktif / 31 hari × Rp3.100.000.
	id, err := s.HitungDraf(ctx, idPekerja, idPeriode, 0, 0)
	if err != nil {
		t.Fatalf("buat draf bulanan: %v", err)
	}

	gaji, err := repo.CariByID(ctx, int(id))
	if err != nil {
		t.Fatal(err)
	}
	if gaji.GajiPokok != "1600000.00" ||
		gaji.TotalLembur != "62716.76" ||
		gaji.PotonganTelat != "17919.08" ||
		gaji.GajiBersih != "1644798.00" {
		t.Fatalf("hasil prorata bulanan tidak sesuai: %+v", gaji)
	}

	// 2. Tarif naik menjadi Rp6.200.000 mulai tanggal 21.
	_, err = db.Exec(`
		INSERT INTO riwayat_upah (id_pekerja, berlaku_mulai, nominal)
		VALUES (?, '2093-01-21', '6200000.00')`, idPekerja)
	if err != nil {
		t.Fatal(err)
	}

	idLagi, err := s.HitungDraf(ctx, idPekerja, idPeriode, 0, 0)
	if err != nil {
		t.Fatalf("hitung ulang tarif bulanan: %v", err)
	}
	if idLagi != id {
		t.Fatal("hitung ulang seharusnya memakai ID gaji yang sama")
	}

	// Tanggal 16–20: 5 × Rp100.000.
	// Tanggal 21–31: 11 × Rp200.000.
	gaji, err = repo.CariByID(ctx, int(id))
	if err != nil {
		t.Fatal(err)
	}
	if gaji.GajiPokok != "2700000.00" ||
		gaji.TotalLembur != "62716.76" ||
		gaji.PotonganTelat != "17919.08" ||
		gaji.GajiBersih != "2744798.00" {
		t.Fatalf("hasil perubahan tarif tidak sesuai: %+v", gaji)
	}

	detail, err := repo.DetailPerGaji(ctx, int(id))
	if err != nil {
		t.Fatal(err)
	}
	if len(detail) != 16 {
		t.Fatalf("ingin 16 detail hari kalender aktif, mendapat %d", len(detail))
	}
	if detail[0].Tanggal.Format("2006-01-02") != "2093-01-16" ||
		detail[15].Tanggal.Format("2006-01-02") != "2093-01-31" {
		t.Fatal("rentang tanggal rincian tidak sesuai masa aktif pekerja")
	}

	for i, d := range detail {
		nominal := "3100000.00"
		pokok := "100000.00"
		if i >= 5 {
			nominal = "6200000.00"
			pokok = "200000.00"
		}

		if d.NominalUpah != nominal || d.NilaiPokok != pokok {
			t.Fatalf("tarif tanggal %s tidak sesuai: %+v",
				d.Tanggal.Format("2006-01-02"), d)
		}
		if i == 0 && d.IDAbsensi == nil {
			t.Fatal("rincian tanggal 16 harus terhubung ke absensi")
		}
		if i > 0 && d.IDAbsensi != nil {
			t.Fatal("hari tanpa jadwal seharusnya tidak memiliki absensi")
		}
	}
}

func TestBayarGajiMySQL(t *testing.T) {
	db := bukaDBDrafUji(t)
	idPekerja, idPeriode, idJadwal := siapkanDataDrafUji(t, db)

	ctx := context.Background()
	repo := &repository.GajiRepo{DB: db}
	s := &GajiService{Repo: repo}

	tolak := func(err error) {
		t.Helper()
		var validasi *ValidasiError
		if !errors.As(err, &validasi) {
			t.Fatalf("ingin error validasi, mendapat: %v", err)
		}
	}

	id, err := s.HitungDraf(ctx, idPekerja, idPeriode, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	// Pemilik tidak berwenang melakukan pembayaran.
	tolak(s.Bayar(ctx, model.PeranPemilik, idPekerja, idPeriode))

	// Periode tertutup menolak pembayaran.
	_, err = db.Exec(`
		UPDATE periode_gaji SET status = 'tertutup'
		WHERE id_periode = ?`, idPeriode)
	if err != nil {
		t.Fatal(err)
	}
	tolak(s.Bayar(ctx, model.PeranAdmin, idPekerja, idPeriode))

	_, err = db.Exec(`
		UPDATE periode_gaji SET status = 'terbuka'
		WHERE id_periode = ?`, idPeriode)
	if err != nil {
		t.Fatal(err)
	}

	// Kasbon yang belum masuk draf tetap harus terdeteksi.
	_, err = db.Exec(`
		INSERT INTO kasbon (id_pekerja, tanggal, jumlah)
		VALUES (?, '2093-01-04', '10000.00')`, idPekerja)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, err := db.Exec(
			"DELETE FROM kasbon WHERE id_pekerja = ?", idPekerja)
		if err != nil {
			t.Errorf("bersihkan kasbon uji: %v", err)
		}
	})

	tolak(s.Bayar(ctx, model.PeranAdmin, idPekerja, idPeriode))

	_, err = db.Exec(
		"DELETE FROM kasbon WHERE id_pekerja = ?", idPekerja)
	if err != nil {
		t.Fatal(err)
	}

	// Perubahan absensi membuat nominal draf sebelumnya kedaluwarsa.
	_, err = db.Exec(`
		UPDATE absensi SET jam_keluar = '2093-01-05 16:00:00'
		WHERE id_jadwal = ?`, idJadwal)
	if err != nil {
		t.Fatal(err)
	}
	tolak(s.Bayar(ctx, model.PeranAdmin, idPekerja, idPeriode))

	if _, err := s.HitungDraf(ctx, idPekerja, idPeriode, 0, 0); err != nil {
		t.Fatal(err)
	}

	// Draf terbaru berhasil dibayar.
	if err := s.Bayar(ctx, model.PeranAdmin, idPekerja, idPeriode); err != nil {
		t.Fatalf("bayar gaji: %v", err)
	}

	gaji, err := repo.CariByID(ctx, int(id))
	if err != nil {
		t.Fatal(err)
	}
	if gaji.StatusBayar != "dibayar" || gaji.DibayarPada == nil {
		t.Fatalf("pembayaran belum tercatat dengan benar: %+v", gaji)
	}

	// Pembayaran kedua dan hitung ulang setelah dibayar harus ditolak.
	tolak(s.Bayar(ctx, model.PeranAdmin, idPekerja, idPeriode))
	_, err = s.HitungDraf(ctx, idPekerja, idPeriode, 0, 0)
	tolak(err)

	setelah, err := repo.CariByID(ctx, int(id))
	if err != nil {
		t.Fatal(err)
	}
	if setelah.DibayarPada == nil ||
		!setelah.DibayarPada.Equal(*gaji.DibayarPada) {
		t.Fatal("waktu pembayaran pertama berubah")
	}
}