package service

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/anggyputra/go-class/internal/model"
	"github.com/anggyputra/go-class/internal/repository"
)

// PekerjaService menangani proses bisnis data pekerja.
type PekerjaService struct {
	Repo *repository.PekerjaRepo
}

// Daftar mengambil seluruh data pekerja.
func (s *PekerjaService) Daftar(ctx context.Context) ([]model.Pekerja, error) {
	return s.Repo.Semua(ctx)
}

// Detail mengambil data pekerja berdasarkan ID.
func (s *PekerjaService) Detail(ctx context.Context, id int) (model.Pekerja, error) {
	if id <= 0 {
		return model.Pekerja{}, &ValidasiError{
			Pesan: "ID pekerja tidak valid",
		}
	}

	return s.Repo.CariByID(ctx, id)
}

// Buat menambahkan pekerja baru setelah validasi.
func (s *PekerjaService) Buat(ctx context.Context, p model.Pekerja) (int64, error) {
	p.Nama = strings.TrimSpace(p.Nama)
	p.JenisPekerja = strings.ToLower(strings.TrimSpace(p.JenisPekerja))
	p.Jabatan = strings.TrimSpace(p.Jabatan)

	if err := validasiPekerja(p); err != nil {
		return 0, err
	}

	// Pekerja baru berstatus aktif.
	p.TanggalKeluar = nil

	return s.Repo.Tambah(ctx, p)
}

// Ubah memperbarui data pekerja setelah validasi.
func (s *PekerjaService) Ubah(ctx context.Context, p model.Pekerja) error {
	if p.ID <= 0 {
		return &ValidasiError{
			Pesan: "ID pekerja tidak valid",
		}
	}

	// Ambil data lama agar status tidak berubah
	// melalui formulir edit biasa.
	lama, err := s.Repo.CariByID(ctx, p.ID)
	if err != nil {
		return err
	}

	p.TanggalKeluar = lama.TanggalKeluar

	p.Nama = strings.TrimSpace(p.Nama)
	p.JenisPekerja = strings.ToLower(strings.TrimSpace(p.JenisPekerja))
	p.Jabatan = strings.TrimSpace(p.Jabatan)

	if err := validasiPekerja(p); err != nil {
		return err
	}

	return s.Repo.Ubah(ctx, p)
}

// Nonaktifkan mencatat tanggal keluar pekerja.
func (s *PekerjaService) Nonaktifkan(ctx context.Context, id int, tanggalKeluar time.Time) error {
	if id <= 0 {
		return &ValidasiError{
			Pesan: "ID pekerja tidak valid",
		}
	}

	if tanggalKeluar.IsZero() {
		return &ValidasiError{
			Pesan: "Tanggal keluar wajib diisi",
		}
	}

	p, err := s.Repo.CariByID(ctx, id)
	if err != nil {
		return err
	}
	if p.TanggalKeluar != nil {
		return &ValidasiError{
			Pesan: "Pekerja sudah berstatus nonaktif",
		}
	}
	if tanggalKeluar.Format("2006-01-02") < p.TanggalMasuk.Format("2006-01-02") {
		return &ValidasiError{
			Pesan: "Tanggal keluar tidak boleh sebelum tanggal masuk",
		}
	}

	return errorAturan(s.Repo.Nonaktifkan(ctx, id, tanggalKeluar))
}

// validasiPekerja memeriksa kelengkapan dan aturan data pekerja.
func validasiPekerja(p model.Pekerja) error {
	nama := strings.TrimSpace(p.Nama)
	if nama == "" {
		return &ValidasiError{
			Pesan: "Nama pekerja wajib diisi",
		}
	}
	if utf8.RuneCountInString(nama) > 100 {
		return &ValidasiError{
			Pesan: "Nama pekerja maksimal 100 karakter",
		}
	}

	if p.JenisPekerja != "harian" && p.JenisPekerja != "bulanan" {
		return &ValidasiError{
			Pesan: "Jenis pekerja harus harian atau bulanan",
		}
	}

	jabatan := strings.TrimSpace(p.Jabatan)
	if jabatan == "" {
		return &ValidasiError{
			Pesan: "Jabatan pekerja wajib diisi",
		}
	}
	if utf8.RuneCountInString(jabatan) > 100 {
		return &ValidasiError{
			Pesan: "Jabatan pekerja maksimal 100 karakter",
		}
	}

	if p.Kontak != nil && utf8.RuneCountInString(strings.TrimSpace(*p.Kontak)) > 50 {
		return &ValidasiError{
			Pesan: "Kontak pekerja maksimal 50 karakter",
		}
	}

	if p.TanggalMasuk.IsZero() {
		return &ValidasiError{
			Pesan: "Tanggal masuk wajib diisi",
		}
	}

	if p.TanggalKeluar != nil {
		tanggalMasuk := p.TanggalMasuk.Format("2006-01-02")
		tanggalKeluar := p.TanggalKeluar.Format("2006-01-02")

		if tanggalKeluar < tanggalMasuk {
			return &ValidasiError{
				Pesan: "Tanggal keluar tidak boleh sebelum tanggal masuk",
			}
		}
	}

	return nil
}
