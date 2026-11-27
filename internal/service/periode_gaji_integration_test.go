package service

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/anggyputra/go-class/internal/model"
	"github.com/anggyputra/go-class/internal/repository"
	"github.com/go-sql-driver/mysql"
)

func TestBuatPeriodeMySQL(t *testing.T) {
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

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	s := &PeriodeGajiService{
		Repo: &repository.PeriodeGajiRepo{DB: db},
	}

	mulai := time.Date(2092, time.November, 1, 0, 0, 0, 0, time.Local)
	akhir := mulai.AddDate(0, 1, -1)

	p := model.PeriodeGaji{
		Jenis:               "bulanan",
		TanggalAwal:         mulai,
		TanggalAkhir:        akhir,
		TanggalBayarRencana: akhir,
	}

	id, err := s.Buat(ctx, p)
	if err != nil {
		t.Fatalf("buat periode: %v", err)
	}

	// Bersihkan periode yang dibuat tes setelah selesai.
	defer func() {
		_, err := db.ExecContext(ctx,
			"DELETE FROM periode_gaji WHERE id_periode = ?", id)
		if err != nil {
			t.Errorf("bersihkan periode uji: %v", err)
		}
	}()

	tersimpan, err := s.Detail(ctx, int(id))
	if err != nil {
		t.Fatalf("ambil periode: %v", err)
	}
	if tersimpan.Jenis != "bulanan" || tersimpan.Status != "terbuka" {
		t.Fatalf("data periode tidak sesuai: %+v", tersimpan)
	}
	if tersimpan.TanggalAwal.Format("2006-01-02") != "2092-11-01" ||
		tersimpan.TanggalAkhir.Format("2006-01-02") != "2092-11-30" {
		t.Fatal("Tanggal periode tersimpan tidak sesuai")
	}

	// Pembuatan kedua dengan periode yang sama harus ditolak.
	idLagi, err := s.Buat(ctx, p)
	if err == nil {
		_, cleanupErr := db.ExecContext(ctx,
			"DELETE FROM periode_gaji WHERE id_periode = ?", idLagi)
		if cleanupErr != nil {
			t.Errorf("bersihkan periode duplikat: %v", cleanupErr)
		}
		t.Fatal("Periode yang sama seharusnya ditolak")
	}

	var validasi *ValidasiError
	if !errors.As(err, &validasi) {
		t.Fatalf("Seharusnya error validasi, mendapat: %v", err)
	}
}
