package service

import (
	"strings"
	"testing"
	"time"

	"github.com/anggyputra/go-class/internal/model"
)

func TestHitungAbsensi(t *testing.T) {
	date := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	j := model.JadwalShift{Tanggal: date, JamMulai: "07:00", JamSelesai: "15:00"}
	jam := func(h, m int) *time.Time { v := time.Date(2026, 1, 5, h, m, 0, 0, time.UTC); return &v }
	besok := time.Date(2026, 1, 6, 1, 0, 0, 0, time.UTC)
	tests := []struct {
		name, status  string
		masuk, keluar *time.Time
		telat, lembur int
		invalid       bool
	}{
		{"15 menit", "hadir", jam(7, 15), jam(15, 0), 15, 0, false},
		{"16 menit dan 30 lembur", "hadir", jam(7, 16), jam(15, 30), 16, 30, false},
		{"datang awal dan 120 lembur", "hadir", jam(6, 30), jam(17, 0), 0, 120, false},
		{"belum lengkap", "perlu_diperiksa", jam(8, 0), nil, 0, 0, false},
		{"hadir belum lengkap", "hadir", jam(8, 0), nil, 0, 0, true},
		{"tidak hadir dengan jam", "tidak_hadir", jam(8, 0), nil, 0, 0, true},
		{"jam terbalik", "hadir", jam(8, 0), jam(7, 0), 0, 0, true},
		{"lintas tanggal", "hadir", jam(8, 0), &besok, 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := model.Absensi{Status: tt.status, JamMasuk: tt.masuk, JamKeluar: tt.keluar}
			err := hitungAbsensi(&a, j)
			if (err != nil) != tt.invalid {
				t.Fatalf("error = %v", err)
			}
			if err == nil && (a.MenitTerlambat != tt.telat || a.MenitLembur != tt.lembur) {
				t.Fatalf("durasi = %d, %d", a.MenitTerlambat, a.MenitLembur)
			}
		})
	}
}

func TestBatasKarakterUnicode(t *testing.T) {
	shift := model.Shift{Nama: strings.Repeat("é", 50), JamMulai: "07:00", JamSelesai: "15:00"}
	if err := validasiShift(shift); err != nil {
		t.Fatalf("50 karakter: %v", err)
	}
	shift.Nama += "é"
	if err := validasiShift(shift); err == nil {
		t.Fatal("51 karakter harus ditolak")
	}
	a := model.Absensi{Status: "perlu_diperiksa", Keterangan: strings.Repeat("é", 255)}
	if err := hitungAbsensi(&a, model.JadwalShift{}); err != nil {
		t.Fatalf("255 karakter: %v", err)
	}
	a.Keterangan += "é"
	if err := hitungAbsensi(&a, model.JadwalShift{}); err == nil {
		t.Fatal("256 karakter harus ditolak")
	}
}
