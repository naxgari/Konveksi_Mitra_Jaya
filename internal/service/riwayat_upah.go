package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"

	"github.com/anggyputra/go-class/internal/model"
	"github.com/anggyputra/go-class/internal/repository"
)

// RiwayatUpahService menangani proses bisnis riwayat upah.
type RiwayatUpahService struct {
	Repo        *repository.RiwayatUpahRepo
	PekerjaRepo *repository.PekerjaRepo
}

// Pola nominal sesuai DECIMAL(15,2) pada MySQL.
var polaNominalUpah = regexp.MustCompile(
	`^(0|[1-9][0-9]{0,12})(\.[0-9]{1,2})?$`,
)

// Daftar mengambil riwayat upah berdasarkan pekerja.
func (s *RiwayatUpahService) Daftar(
	ctx context.Context,
	idPekerja int,
) ([]model.RiwayatUpah, error) {

	if idPekerja <= 0 {
		return nil, &ValidasiError{
			Pesan: "ID pekerja tidak valid",
		}
	}

	// Pastikan pekerja tersedia.
	_, err := s.PekerjaRepo.CariByID(ctx, idPekerja)
	if err != nil {
		return nil, err
	}

	return s.Repo.SemuaByPekerja(ctx, idPekerja)
}

// Buat menambahkan riwayat tarif upah baru.
func (s *RiwayatUpahService) Buat(
	ctx context.Context,
	u model.RiwayatUpah,
) (int64, error) {

	// 1. Validasi ID Pekerja

	if u.IDPekerja <= 0 {
		return 0, &ValidasiError{
			Pesan: "ID pekerja tidak valid",
		}
	}

	// 2. Validasi Tanggal Berlaku

	if u.BerlakuMulai.IsZero() {
		return 0, &ValidasiError{
			Pesan: "Tanggal mulai berlaku wajib diisi",
		}
	}

	// 3. Validasi Nominal Upah

	u.Nominal = strings.TrimSpace(u.Nominal)

	if !polaNominalUpah.MatchString(u.Nominal) {
		return 0, &ValidasiError{
			Pesan: "Nominal harus berupa angka positif dengan maksimal 2 angka desimal",
		}
	}

	// Nominal tidak boleh nol.
	if u.Nominal == "0" ||
		u.Nominal == "0.0" ||
		u.Nominal == "0.00" {

		return 0, &ValidasiError{
			Pesan: "Nominal upah harus lebih dari nol",
		}
	}

	// 4. Memastikan Pekerja Tersedia

	p, err := s.PekerjaRepo.CariByID(
		ctx,
		u.IDPekerja,
	)

	if err != nil {
		return 0, err
	}

	tanggalBerlaku := u.BerlakuMulai.Format("2006-01-02")
	tanggalMasuk := p.TanggalMasuk.Format("2006-01-02")

	// Tarif tidak boleh berlaku sebelum pekerja masuk.
	if tanggalBerlaku < tanggalMasuk {
		return 0, &ValidasiError{
			Pesan: "Tanggal berlaku tidak boleh sebelum tanggal masuk pekerja",
		}
	}

	// 5. Validasi Tanggal Keluar

	if p.TanggalKeluar != nil {
		tanggalKeluar := p.TanggalKeluar.Format("2006-01-02")

		if tanggalBerlaku > tanggalKeluar {
			return 0, &ValidasiError{
				Pesan: "Tanggal berlaku tidak boleh setelah pekerja keluar",
			}
		}
	}

	// 6. Validasi Keterangan

	if u.Keterangan != nil {
		ket := strings.TrimSpace(*u.Keterangan)

		if utf8.RuneCountInString(ket) > 255 {
			return 0, &ValidasiError{
				Pesan: "Keterangan maksimal 255 karakter",
			}
		}

		if ket == "" {
			u.Keterangan = nil
		} else {
			u.Keterangan = &ket
		}
	}

	// 7. Simpan Riwayat Upah

	id, err := s.Repo.Tambah(ctx, u)

	if err != nil {

		// Periksa apakah database menolak data duplikat.
		var mysqlErr *mysql.MySQLError

		if errors.As(err, &mysqlErr) {
			if mysqlErr.Number == 1062 {
				return 0, &ValidasiError{
					Pesan: "Riwayat upah pada tanggal tersebut sudah ada. Silakan gunakan tanggal lain.",
				}
			}
		}

		// Error lain diteruskan seperti sebelumnya.
		return 0, err
	}

	return id, nil
}

// TarifBerlaku mengambil tarif yang berlaku
// pada tanggal yang diminta.
func (s *RiwayatUpahService) TarifBerlaku(
	ctx context.Context,
	idPekerja int,
	tanggal time.Time,
) (model.RiwayatUpah, error) {

	// 8. Validasi Parameter Tarif Berlaku

	if idPekerja <= 0 {
		return model.RiwayatUpah{}, &ValidasiError{
			Pesan: "ID pekerja tidak valid",
		}
	}

	if tanggal.IsZero() {
		return model.RiwayatUpah{}, &ValidasiError{
			Pesan: "Tanggal perhitungan wajib diisi",
		}
	}

	// Mengambil tarif yang berlaku dari repository.
	return s.Repo.TarifBerlaku(
		ctx,
		idPekerja,
		tanggal,
	)
}
