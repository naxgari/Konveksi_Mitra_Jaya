package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/anggyputra/go-class/internal/config"
	"github.com/anggyputra/go-class/internal/handler"
	"github.com/anggyputra/go-class/internal/middleware"
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

	// Modul Contoh Bawaan Template (Produk)
	produkRepo := &repository.ProdukRepo{DB: db}
	produkSvc := &service.ProdukService{Repo: produkRepo}
	produk := &handler.ProdukHandler{Svc: produkSvc, View: view}

	// Private ServeMux: Seluruh route di sini wajib Login (RequireLogin)
	privateMux := http.NewServeMux()
	privateMux.HandleFunc("GET /{$}", halaman.Home)
	privateMux.HandleFunc("GET /profil", halaman.Profil)
	privateMux.HandleFunc("POST /logout", auth.Logout)

	// Route Contoh Produk (Referensi Anggota Tim)
	privateMux.HandleFunc("GET /produk", produk.Daftar)
	privateMux.HandleFunc("GET /produk/form", produk.FormTambah)
	privateMux.HandleFunc("POST /produk", produk.Simpan)
	privateMux.HandleFunc("GET /produk/{id}", produk.Detail)
	privateMux.HandleFunc("GET /produk/{id}/edit", produk.FormEdit)
	privateMux.HandleFunc("POST /produk/{id}/edit", produk.Update)
	privateMux.HandleFunc("POST /produk/{id}/hapus", produk.Hapus)

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
		Handler:           middleware.Logging(rootMux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("server Konveksi Mitra Jaya jalan di %s", cfg.Addr)
	log.Fatal(srv.ListenAndServe())
}

func bukaDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	log.Println("database terhubung")
	return db, nil
}
