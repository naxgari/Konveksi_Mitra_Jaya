package service

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/anggyputra/go-class/internal/model"
	"github.com/anggyputra/go-class/internal/repository"
	_ "github.com/go-sql-driver/mysql"
)

func TestAbsensiMySQL(t *testing.T) {
	dsn := os.Getenv("FALIH_TEST_DSN")
	if dsn == "" {
		t.Skip("FALIH_TEST_DSN tidak disetel")
	}
	db, e := sql.Open("mysql", dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	ctx := context.Background()
	exec := func(q string, args ...any) sql.Result {
		t.Helper()
		r, e := db.ExecContext(ctx, q, args...)
		if e != nil {
			t.Fatal(e)
		}
		return r
	}
	id := func(r sql.Result) int {
		v, e := r.LastInsertId()
		if e != nil {
			t.Fatal(e)
		}
		return int(v)
	}
	harian := id(exec(`INSERT INTO pekerja(nama,jenis_pekerja,jabatan,tanggal_masuk,tanggal_keluar) VALUES('Uji Falih Harian','harian','Uji','2091-01-01','2091-12-31')`))
	bulanan := id(exec(`INSERT INTO pekerja(nama,jenis_pekerja,jabatan,tanggal_masuk,tanggal_keluar) VALUES('Uji Falih Bulanan','bulanan','Uji','2091-01-01','2091-12-31')`))
	shift := id(exec(`INSERT INTO shift(nama_shift,jam_mulai,jam_selesai) VALUES('Uji Falih Shift','07:00','15:00')`))
	var periodeIDs []int
	defer func() {
		db.Exec(`DELETE a FROM absensi a JOIN jadwal_shift j ON j.id_jadwal=a.id_jadwal WHERE j.id_pekerja IN (?,?)`, harian, bulanan)
		db.Exec(`DELETE FROM jadwal_shift WHERE id_pekerja IN (?,?)`, harian, bulanan)
		db.Exec(`DELETE FROM gaji WHERE id_pekerja IN (?,?)`, harian, bulanan)
		for _, pid := range periodeIDs {
			db.Exec(`DELETE FROM periode_gaji WHERE id_periode=?`, pid)
		}
		db.Exec(`DELETE FROM pekerja WHERE id_pekerja IN (?,?)`, harian, bulanan)
		db.Exec(`DELETE FROM shift WHERE id_shift=?`, shift)
	}()
	s := AbsensiService{Repo: &repository.AbsensiRepo{DB: db}}
	date := func(s string) time.Time {
		v, e := time.Parse("2006-01-02", s)
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	unused := model.Shift{Nama: "Uji Falih Sementara", JamMulai: "08:00", JamSelesai: "16:00"}
	unusedID, e := s.ShiftBuat(ctx, unused)
	if e != nil {
		t.Fatal(e)
	}
	unused.ID = int(unusedID)
	defer db.Exec(`DELETE FROM shift WHERE id_shift=?`, unused.ID)
	if e = s.ShiftUbah(ctx, unused); e != nil {
		t.Fatalf("ubah shift tanpa perubahan: %v", e)
	}
	unused.Nama = "Uji Falih Diubah"
	if e = s.ShiftUbah(ctx, unused); e != nil {
		t.Fatal(e)
	}
	if e = s.ShiftHapus(ctx, unused.ID); e != nil {
		t.Fatal(e)
	}
	for _, tanggal := range []string{"2091-01-01", "2091-12-31"} {
		boundary := model.JadwalShift{IDPekerja: harian, IDShift: shift, Tanggal: date(tanggal)}
		if _, e = s.JadwalBuat(ctx, boundary); e != nil {
			t.Fatalf("tanggal batas %s: %v", tanggal, e)
		}
	}
	j := model.JadwalShift{IDPekerja: harian, IDShift: shift, Tanggal: date("2091-02-05")}
	expectInvalid := func(e error) {
		t.Helper()
		var v *ValidasiError
		if !errors.As(e, &v) {
			t.Fatalf("diharapkan ValidasiError, dapat %v", e)
		}
	}
	jid, e := s.JadwalBuat(ctx, j)
	if e != nil {
		t.Fatal(e)
	}
	j.ID = int(jid)
	if e = s.JadwalUbah(ctx, j); e != nil {
		t.Fatalf("ubah jadwal tanpa perubahan: %v", e)
	}
	_, e = s.JadwalBuat(ctx, j)
	expectInvalid(e)
	invalid := j
	invalid.Tanggal = date("2092-01-01")
	_, e = s.JadwalBuat(ctx, invalid)
	expectInvalid(e)
	expectInvalid(s.ShiftUbah(ctx, model.Shift{ID: shift, Nama: "Diubah", JamMulai: "07:00", JamSelesai: "15:00"}))
	masuk := date("2091-02-05").Add(7*time.Hour + 16*time.Minute)
	keluar := date("2091-02-05").Add(15*time.Hour + 30*time.Minute)
	incomplete := model.Absensi{IDJadwal: j.ID, Status: "perlu_diperiksa", JamMasuk: &masuk}
	if e = s.AbsensiSimpan(ctx, incomplete); e != nil {
		t.Fatal(e)
	}
	incompleteGot, e := s.JadwalDetail(ctx, j.ID)
	if e != nil {
		t.Fatal(e)
	}
	if incompleteGot.Absensi == nil || incompleteGot.Absensi.Status != "perlu_diperiksa" || incompleteGot.Absensi.JamKeluar != nil || incompleteGot.Absensi.MenitTerlambat != 0 {
		t.Fatalf("absensi belum lengkap: %+v", incompleteGot.Absensi)
	}
	a := model.Absensi{IDJadwal: j.ID, Status: "hadir", JamMasuk: &masuk, JamKeluar: &keluar}
	if e = s.AbsensiSimpan(ctx, a); e != nil {
		t.Fatal(e)
	}
	if e = s.AbsensiSimpan(ctx, a); e != nil {
		t.Fatalf("simpan absensi tanpa perubahan: %v", e)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- s.AbsensiSimpan(ctx, a) }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("simpan bersamaan: %v", err)
		}
	}
	var jumlah int
	if e = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM absensi WHERE id_jadwal=?`, j.ID).Scan(&jumlah); e != nil || jumlah != 1 {
		t.Fatalf("jumlah absensi = %d, error = %v", jumlah, e)
	}
	got, e := s.JadwalDetail(ctx, j.ID)
	if e != nil {
		t.Fatal(e)
	}
	if got.Absensi == nil || got.Absensi.MenitTerlambat != 16 || got.Absensi.MenitLembur != 30 {
		t.Fatalf("hasil absensi: %+v", got.Absensi)
	}
	expectInvalid(s.JadwalHapus(ctx, j.ID))
	periodeIDs = append(periodeIDs, id(exec(`INSERT INTO periode_gaji(jenis_periode,tanggal_awal,tanggal_akhir,tanggal_bayar_rencana,status) VALUES('mingguan','2091-02-05','2091-02-10','2091-02-10','tertutup')`)))
	expectInvalid(s.AbsensiSimpan(ctx, a))
	j2 := model.JadwalShift{IDPekerja: bulanan, IDShift: shift, Tanggal: j.Tanggal}
	if _, e = s.JadwalBuat(ctx, j2); e != nil {
		t.Fatalf("mingguan tidak boleh mengunci bulanan: %v", e)
	}
	pid := id(exec(`INSERT INTO periode_gaji(jenis_periode,tanggal_awal,tanggal_akhir,tanggal_bayar_rencana,status) VALUES('bulanan','2091-02-01','2091-02-28','2091-02-28','terbuka')`))
	periodeIDs = append(periodeIDs, pid)
	exec(`INSERT INTO gaji(id_pekerja,id_periode,status_bayar) VALUES(?,?,'dibayar')`, bulanan, pid)
	j2.Tanggal = date("2091-02-06")
	_, e = s.JadwalBuat(ctx, j2)
	expectInvalid(e)
}
