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
	halaman := &handler.HalamanHandler{View: view}

	produkRepo := &repository.ProdukRepo{DB: db}
	produkSvc := &service.ProdukService{Repo: produkRepo}
	produk := &handler.ProdukHandler{Svc: produkSvc, View: view}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", halaman.Home)
	mux.HandleFunc("GET /profil", halaman.Profil)

	mux.HandleFunc("GET /produk", produk.Daftar)
	mux.HandleFunc("GET /produk/form", produk.FormTambah)
	mux.HandleFunc("POST /produk", produk.Simpan)
	mux.HandleFunc("GET /produk/{id}", produk.Detail)
	mux.HandleFunc("GET /produk/{id}/edit", produk.FormEdit)
	mux.HandleFunc("POST /produk/{id}/edit", produk.Update)
	mux.HandleFunc("POST /produk/{id}/hapus", produk.Hapus)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           middleware.Logging(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("server jalan di %s", cfg.Addr)
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
