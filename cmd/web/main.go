package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"

	"github.com/anggyputra/go-class/internal/config"
	"github.com/anggyputra/go-class/internal/handler"
	"github.com/anggyputra/go-class/internal/middleware"
	"github.com/anggyputra/go-class/internal/model"
	"github.com/anggyputra/go-class/internal/repository"
	"github.com/anggyputra/go-class/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := bukaDB(cfg.DBDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	view := handler.NewRenderer()

	// 1. Modul Autentikasi & Pengguna (Anggota 1 - Naufal)
	penggunaRepo := &repository.PenggunaRepo{DB: db}
	authSvc := &service.AuthService{Repo: penggunaRepo}
	authSvc.SeedPenggunaAwal(context.Background())

	auth := &handler.AuthHandler{
		Svc:          authSvc,
		View:         view,
		CookieSecure: cfg.CookieSecure,
	}
	halaman := &handler.HalamanHandler{
		View:    view,
		AuthSvc: authSvc,
	}

	// =====================================================
	// 3. Modul Master Pekerja (Cahya)
	// =====================================================

	pekerjaRepo := &repository.PekerjaRepo{DB: db}
	pekerjaSvc := &service.PekerjaService{Repo: pekerjaRepo}
	pekerja := &handler.PekerjaHandler{
		Svc:  pekerjaSvc,
		View: view,
	}

	// =====================================================
	// 4. Modul Riwayat Upah (Cahya)
	// =====================================================

	upahRepo := &repository.RiwayatUpahRepo{DB: db}

	upahSvc := &service.RiwayatUpahService{
		Repo:        upahRepo,
		PekerjaRepo: pekerjaRepo,
	}

	upah := &handler.RiwayatUpahHandler{
		Svc:        upahSvc,
		PekerjaSvc: pekerjaSvc,
		View:       view,
	}

	// =====================================================
	// 5. Modul Jadwal Shift & Absensi (Falih)
	// =====================================================

	absensiRepo := &repository.AbsensiRepo{DB: db}
	absensiSvc := &service.AbsensiService{Repo: absensiRepo}
	absensi := &handler.AbsensiHandler{Svc: absensiSvc, View: view}

	// Modul Penggajian milik Daffa.
	periodeRepo := &repository.PeriodeGajiRepo{DB: db}
	periodeSvc := &service.PeriodeGajiService{Repo: periodeRepo}

	gajiRepo := &repository.GajiRepo{DB: db}
	gajiSvc := &service.GajiService{Repo: gajiRepo}

	penggajian := &handler.PenggajianHandler{
		PeriodeSvc: periodeSvc,
		GajiSvc:    gajiSvc,
		View:       view,
	}

	kasbonRepo := &repository.KasbonRepo{DB: db}
	kasbonSvc := &service.KasbonService{Repo: kasbonRepo}
	kasbon := &handler.KasbonHandler{Svc: kasbonSvc, GajiSvc: gajiSvc, View: view}

	// Private ServeMux: Seluruh route di sini wajib Login (RequireLogin)
	privateMux := http.NewServeMux()
	privateMux.HandleFunc("GET /{$}", halaman.Home)
	privateMux.HandleFunc("GET /profil", halaman.Profil)

	// Modul Master Pekerja & Riwayat Upah (Cahya)
	privateMux.HandleFunc("GET /pekerja", middleware.RequireRole(model.PeranAdmin, model.PeranPemilik)(pekerja.Daftar))
	privateMux.HandleFunc("GET /pekerja/form", middleware.RequireRole(model.PeranAdmin)(pekerja.FormTambah))
	privateMux.HandleFunc("POST /pekerja", middleware.RequireRole(model.PeranAdmin)(pekerja.Simpan))
	privateMux.HandleFunc("GET /pekerja/{id}", middleware.RequireRole(model.PeranAdmin, model.PeranPemilik)(pekerja.Detail))
	privateMux.HandleFunc("GET /pekerja/{id}/edit", middleware.RequireRole(model.PeranAdmin)(pekerja.FormEdit))
	privateMux.HandleFunc("POST /pekerja/{id}/edit", middleware.RequireRole(model.PeranAdmin)(pekerja.Update))
	privateMux.HandleFunc("POST /pekerja/{id}/nonaktif", middleware.RequireRole(model.PeranAdmin)(pekerja.Nonaktifkan))
	privateMux.HandleFunc("GET /pekerja/{id}/upah", middleware.RequireRole(model.PeranAdmin, model.PeranPemilik)(upah.Daftar))
	privateMux.HandleFunc("GET /pekerja/{id}/upah/form", middleware.RequireRole(model.PeranAdmin)(upah.FormTambah))
	privateMux.HandleFunc("POST /pekerja/{id}/upah", middleware.RequireRole(model.PeranAdmin)(upah.Simpan))

	// Modul Kasbon, Koreksi Gaji, Slip & Laporan Rekap (Adel)
	privateMux.HandleFunc("GET /kasbon", middleware.RequireRole(model.PeranAdmin, model.PeranPemilik)(kasbon.Daftar))
	privateMux.HandleFunc("GET /kasbon/form", middleware.RequireRole(model.PeranAdmin)(kasbon.Form))
	privateMux.HandleFunc("POST /kasbon", middleware.RequireRole(model.PeranAdmin)(kasbon.Simpan))
	privateMux.HandleFunc("GET /kasbon/{id}", middleware.RequireRole(model.PeranAdmin, model.PeranPemilik)(kasbon.Detail))
	privateMux.HandleFunc("POST /kasbon/{id}/hapus", middleware.RequireRole(model.PeranAdmin)(kasbon.Hapus))
	privateMux.HandleFunc("GET /koreksi", middleware.RequireRole(model.PeranAdmin, model.PeranPemilik)(kasbon.KoreksiDaftar))
	privateMux.HandleFunc("GET /koreksi/form", middleware.RequireRole(model.PeranAdmin)(kasbon.KoreksiForm))
	privateMux.HandleFunc("POST /koreksi", middleware.RequireRole(model.PeranAdmin)(kasbon.KoreksiSimpan))
	privateMux.HandleFunc("POST /koreksi/{id}/setujui", middleware.RequireRole(model.PeranPemilik)(kasbon.KoreksiSetujui))
	privateMux.HandleFunc("POST /koreksi/{id}/hapus", middleware.RequireRole(model.PeranAdmin)(kasbon.KoreksiHapus))
	privateMux.HandleFunc("GET /gaji/{id}/slip", middleware.RequireRole(model.PeranAdmin, model.PeranPemilik)(kasbon.Slip))
	privateMux.HandleFunc("POST /gaji/{id}/hitung-komponen", middleware.RequireRole(model.PeranAdmin)(kasbon.HitungKomponenTambahan))
	privateMux.HandleFunc("GET /laporan/rekap", middleware.RequireRole(model.PeranAdmin, model.PeranPemilik)(kasbon.Rekap))

	// Modul Shift & Absensi (Falih)
	privateMux.HandleFunc("GET /jadwal", middleware.RequireRole(model.PeranAdmin, model.PeranPemilik)(absensi.JadwalDaftar))
	privateMux.HandleFunc("GET /jadwal/form", middleware.RequireRole(model.PeranAdmin)(absensi.JadwalFormTambah))
	privateMux.HandleFunc("POST /jadwal", middleware.RequireRole(model.PeranAdmin)(absensi.JadwalSimpan))
	privateMux.HandleFunc("GET /jadwal/{id}/edit", middleware.RequireRole(model.PeranAdmin)(absensi.JadwalFormEdit))
	privateMux.HandleFunc("POST /jadwal/{id}/edit", middleware.RequireRole(model.PeranAdmin)(absensi.JadwalUpdate))
	privateMux.HandleFunc("POST /jadwal/{id}/hapus", middleware.RequireRole(model.PeranAdmin)(absensi.JadwalHapus))
	privateMux.HandleFunc("GET /jadwal/{id}/absensi", middleware.RequireRole(model.PeranAdmin)(absensi.AbsensiForm))
	privateMux.HandleFunc("POST /jadwal/{id}/absensi", middleware.RequireRole(model.PeranAdmin)(absensi.AbsensiSimpan))
	privateMux.HandleFunc("GET /absensi/pemeriksaan", middleware.RequireRole(model.PeranAdmin, model.PeranPemilik)(absensi.Pemeriksaan))
	privateMux.HandleFunc("GET /shift", middleware.RequireRole(model.PeranAdmin, model.PeranPemilik)(absensi.ShiftDaftar))
	privateMux.HandleFunc("GET /shift/form", middleware.RequireRole(model.PeranAdmin)(absensi.ShiftFormTambah))
	privateMux.HandleFunc("POST /shift", middleware.RequireRole(model.PeranAdmin)(absensi.ShiftSimpan))
	privateMux.HandleFunc("GET /shift/{id}/edit", middleware.RequireRole(model.PeranAdmin)(absensi.ShiftFormEdit))
	privateMux.HandleFunc("POST /shift/{id}/edit", middleware.RequireRole(model.PeranAdmin)(absensi.ShiftUpdate))
	privateMux.HandleFunc("POST /shift/{id}/hapus", middleware.RequireRole(model.PeranAdmin)(absensi.ShiftHapus))

	// Modul Periode & Perhitungan Gaji (Daffa)
	privateMux.HandleFunc("GET /periode", middleware.RequireRole(model.PeranAdmin, model.PeranPemilik)(penggajian.PeriodeDaftar))
	privateMux.HandleFunc("GET /periode/form", middleware.RequireRole(model.PeranAdmin)(penggajian.PeriodeFormTambah))
	privateMux.HandleFunc("POST /periode", middleware.RequireRole(model.PeranAdmin)(penggajian.PeriodeSimpan))
	privateMux.HandleFunc("GET /periode/{id}", middleware.RequireRole(model.PeranAdmin, model.PeranPemilik)(penggajian.PeriodeDetail))
	privateMux.HandleFunc("POST /periode/{id}/hitung", middleware.RequireRole(model.PeranAdmin)(penggajian.HitungDraf))
	privateMux.HandleFunc("POST /periode/{id}/tutup", middleware.RequireRole(model.PeranPemilik)(penggajian.TutupPeriode))
	privateMux.HandleFunc("GET /gaji/{id}", middleware.RequireRole(model.PeranAdmin, model.PeranPemilik)(penggajian.GajiDetail))
	privateMux.HandleFunc("POST /gaji/{id}/bayar", middleware.RequireRole(model.PeranAdmin)(penggajian.BayarGaji))
	privateMux.HandleFunc("POST /logout", auth.Logout)

	// Public & Root ServeMux: Memisahkan route publik (/login, /register) dan route privat
	rootMux := http.NewServeMux()
	rootMux.HandleFunc("GET /login", auth.FormLogin)
	rootMux.HandleFunc("POST /login", auth.ProsesLogin)
	rootMux.HandleFunc("GET /register", auth.FormRegister)
	rootMux.HandleFunc("POST /register", auth.ProsesRegister)

	// Semua route selain /login dan /register diarahkan ke privateMux yang dilindungi RequireLogin
	rootMux.Handle("/", middleware.RequireLogin(authSvc, cfg.CookieSecure, privateMux))

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           middleware.Logging(http.NewCrossOriginProtection().Handler(rootMux)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("server Konveksi Mitra Jaya jalan di %s", cfg.Addr)
	log.Fatal(srv.ListenAndServe())
}

func bukaDB(dsn string) (*sql.DB, error) {
	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		return nil, err
	}
	cfg.ParseTime = true
	if cfg.Loc == nil || cfg.Loc == time.UTC {
		cfg.Loc = time.Local
	}

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	log.Println("database terhubung")
	return db, nil
}
