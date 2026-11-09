package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"

	"github.com/anggyputra/go-class/internal/model"
	"github.com/anggyputra/go-class/internal/repository"
)

const DurasiSesiLogin = 24 * time.Hour

type RegisterInput struct {
	Username           string
	NamaLengkap        string
	Password           string
	KonfirmasiPassword string
	Peran              string
}

type AuthService struct {
	Repo *repository.PenggunaRepo
}

func (s *AuthService) Register(ctx context.Context, in RegisterInput) (int64, error) {
	username := strings.ToLower(strings.TrimSpace(in.Username))
	nama := strings.TrimSpace(in.NamaLengkap)
	peran := model.PeranPengguna(strings.ToLower(strings.TrimSpace(in.Peran)))

	if len(username) < 4 {
		return 0, &ValidasiError{"Username minimal harus 4 karakter"}
	}
	if strings.ContainsAny(username, " \t\n\r") {
		return 0, &ValidasiError{"Username tidak boleh mengandung spasi"}
	}
	if nama == "" {
		return 0, &ValidasiError{"Nama lengkap wajib diisi"}
	}
	if err := validasiPasswordKuat(in.Password); err != nil {
		return 0, err
	}
	if in.Password != in.KonfirmasiPassword {
		return 0, &ValidasiError{"Konfirmasi password tidak cocok"}
	}
	if !peranValid(peran) {
		return 0, &ValidasiError{"Peran pengguna tidak valid (pilih: admin, supervisor, atau pemilik)"}
	}

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("hash password: %w", err)
	}

	u := model.Pengguna{
		Username:     username,
		PasswordHash: string(hashBytes),
		NamaLengkap:  nama,
		Peran:        peran,
		Aktif:        true,
	}

	id, err := s.Repo.TambahPengguna(ctx, u)
	if err != nil {
		if errors.Is(err, model.ErrUsernameSudahDipakai) {
			return 0, &ValidasiError{err.Error()}
		}
		return 0, err
	}
	return id, nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (model.Sesi, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if username == "" || password == "" {
		return model.Sesi{}, &ValidasiError{"Username dan password wajib diisi"}
	}

	u, err := s.Repo.CariByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return model.Sesi{}, model.ErrKredensialSalah
		}
		return model.Sesi{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return model.Sesi{}, model.ErrKredensialSalah
	}

	token, err := buatTokenAcak()
	if err != nil {
		return model.Sesi{}, err
	}

	sesi := model.Sesi{
		Token:      token,
		IDPengguna: u.ID,
		ExpiredAt:  time.Now().Add(DurasiSesiLogin),
		Pengguna:   u,
	}

	if err := s.Repo.SimpanSesi(ctx, sesi); err != nil {
		return model.Sesi{}, err
	}

	// Bersihkan sesi yang sudah kedaluwarsa secara pasif
	_ = s.Repo.HapusSesiKedaluwarsa(ctx)

	return sesi, nil
}

func (s *AuthService) VerifikasiSesi(ctx context.Context, token string) (model.Pengguna, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return model.Pengguna{}, model.ErrSesiTidakValid
	}
	sesi, err := s.Repo.CariSesiValid(ctx, token)
	if err != nil {
		return model.Pengguna{}, err
	}
	return sesi.Pengguna, nil
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	return s.Repo.HapusSesi(ctx, token)
}

func (s *AuthService) RingkasanDashboard(ctx context.Context) model.RingkasanDashboard {
	return s.Repo.AmbilRingkasanDashboard(ctx)
}

// SeedPenggunaAwal otomatis membuat 3 akun default (admin, supervisor, pemilik) jika tabel pengguna masih kosong.
func (s *AuthService) SeedPenggunaAwal(ctx context.Context) {
	jumlah, err := s.Repo.HitungPengguna(ctx)
	if err != nil {
		// Tabel pengguna mungkin belum di-import dari db/schema.sql; lewati tanpa mematikan server.
		log.Printf("info seeder pengguna: %v", err)
		return
	}
	if jumlah > 0 {
		return
	}

	akunDefault := []struct {
		Username    string
		NamaLengkap string
		Password    string
		Peran       model.PeranPengguna
	}{
		{
			Username:    "admin",
			NamaLengkap: "Muhammad Naufal A. (Admin Penggajian)",
			Password:    "password123",
			Peran:       model.PeranAdmin,
		},
		{
			Username:    "supervisor",
			NamaLengkap: "Muhammad Falih A. (Petugas Absensi)",
			Password:    "password123",
			Peran:       model.PeranSupervisor,
		},
		{
			Username:    "pemilik",
			NamaLengkap: "Bapak H. Hendra (Pemilik Konveksi)",
			Password:    "password123",
			Peran:       model.PeranPemilik,
		},
	}

	for _, ak := range akunDefault {
		hashBytes, err := bcrypt.GenerateFromPassword([]byte(ak.Password), bcrypt.DefaultCost)
		if err != nil {
			continue
		}
		_, err = s.Repo.TambahPengguna(ctx, model.Pengguna{
			Username:     ak.Username,
			PasswordHash: string(hashBytes),
			NamaLengkap:  ak.NamaLengkap,
			Peran:        ak.Peran,
			Aktif:        true,
		})
		if err != nil {
			log.Printf("gagal seed akun %s: %v", ak.Username, err)
		}
	}
	log.Println("seeder: 3 akun awal (admin, supervisor, pemilik) berhasil dibuat dengan password: password123")
}

func validasiPasswordKuat(pw string) error {
	if len(pw) < 8 {
		return &ValidasiError{"Password minimal harus 8 karakter"}
	}
	adaHuruf := false
	adaAngka := false
	for _, ch := range pw {
		if unicode.IsLetter(ch) {
			adaHuruf = true
		}
		if unicode.IsDigit(ch) {
			adaAngka = true
		}
	}
	if !adaHuruf || !adaAngka {
		return &ValidasiError{"Password harus mengandung kombinasi huruf dan angka"}
	}
	return nil
}

func peranValid(p model.PeranPengguna) bool {
	return p == model.PeranAdmin || p == model.PeranSupervisor || p == model.PeranPemilik
}

func buatTokenAcak() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("buat token sesi: %w", err)
	}
	return hex.EncodeToString(b), nil
}

