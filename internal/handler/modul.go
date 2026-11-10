package handler

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/anggyputra/go-class/internal/middleware"
)

type ModulPage struct {
	Judul, Subjudul, Pesan, Aksi string
	Filter                       []string
	Kolom                        []string
	Baris                        [][]string
	Metrik                       []ModulMetric
	Admin                        bool
}
type ModulMetric struct{ Label, Nilai string }
type ModulHandler struct {
	DB   *sql.DB
	View *Renderer
}

// Halaman modul menampilkan data yang sudah ada. Mutasi belum diaktifkan sampai
// aturan bisnis dan transaksi masing-masing modul tersedia.
func (h *ModulHandler) Tampil(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.PenggunaDariContext(r.Context())
	page := ModulPage{Admin: user.AdalahAdmin()}
	var query string
	switch r.URL.Path {
	case "/pekerja":
		page.Judul = "Data Pekerja"
		page.Subjudul = "Data induk pekerja · status masa kerja mengikuti tanggal masuk dan keluar"
		page.Aksi = "Tambah Pekerja"
		page.Filter = []string{"Cari pekerja", "Jenis pekerja", "Masa kerja"}
		page.Kolom = []string{"Pekerja", "Jenis", "Jabatan", "Tanggal masuk", "Masa kerja"}
		page.Pesan = "Pekerja dengan riwayat gaji tetap disimpan. Catat tanggal keluar saat pekerja berhenti."
		query = `SELECT nama, jenis_pekerja, jabatan, DATE_FORMAT(tanggal_masuk,'%d %b %Y'), IF(tanggal_masuk > CURDATE(),'Belum aktif',IF(tanggal_keluar IS NULL OR tanggal_keluar >= CURDATE(),'Aktif','Berhenti')) FROM pekerja ORDER BY nama`
	case "/jadwal":
		page.Judul = "Shift dan Jadwal"
		page.Subjudul = "Kelola penempatan shift pekerja"
		page.Aksi = "Tempatkan Shift"
		page.Filter = []string{"Rentang tanggal", "Cari pekerja", "Shift"}
		page.Kolom = []string{"Pekerja", "Tanggal", "Shift", "Jam kerja"}
		page.Metrik = []ModulMetric{{"Shift Pagi", "07.00–15.00"}, {"Shift Siang", "15.00–23.00"}}
		page.Pesan = "Penempatan hanya untuk tanggal dalam masa aktif. Satu pekerja tidak memiliki dua jadwal pada tanggal yang sama."
		query = `SELECT p.nama, DATE_FORMAT(j.tanggal,'%d %b %Y'), s.nama_shift, CONCAT(TIME_FORMAT(s.jam_mulai,'%H.%i'),'–',TIME_FORMAT(s.jam_selesai,'%H.%i')) FROM jadwal_shift j JOIN pekerja p ON p.id_pekerja=j.id_pekerja JOIN shift s ON s.id_shift=j.id_shift ORDER BY j.tanggal DESC,p.nama`
	case "/absensi/pemeriksaan":
		page.Judul = "Absensi"
		page.Subjudul = "Pencatatan dan pemeriksaan kehadiran"
		page.Filter = []string{"Tanggal", "Shift", "Jenis", "Kondisi catatan"}
		page.Kolom = []string{"Pekerja", "Shift", "Kehadiran", "Masuk", "Keluar", "Telat", "Lembur"}
		page.Pesan = "Belum dicatat berarti belum ada absensi. Belum lengkap berarti hadir dengan salah satu jam belum terisi."
		query = `SELECT p.nama,s.nama_shift,CASE WHEN a.id_absensi IS NULL THEN 'Belum dicatat' WHEN a.status_kehadiran='tidak_hadir' THEN 'Tidak hadir' WHEN a.status_kehadiran='perlu_diperiksa' OR a.jam_masuk IS NULL OR a.jam_keluar IS NULL THEN 'Belum lengkap' ELSE 'Hadir' END,COALESCE(TIME_FORMAT(a.jam_masuk,'%H.%i'),'—'),COALESCE(TIME_FORMAT(a.jam_keluar,'%H.%i'),'—'),IF(a.id_absensi IS NULL,'—',CONCAT(a.menit_terlambat,' mnt')),IF(a.id_absensi IS NULL,'—',CONCAT(a.menit_lembur,' mnt')) FROM jadwal_shift j JOIN pekerja p ON p.id_pekerja=j.id_pekerja JOIN shift s ON s.id_shift=j.id_shift LEFT JOIN absensi a ON a.id_jadwal=j.id_jadwal ORDER BY j.tanggal DESC,p.nama`
	case "/kasbon":
		page.Judul = "Kasbon"
		page.Subjudul = "Pengambilan, pemotongan, dan saldo kasbon"
		page.Aksi = "Catat Kasbon"
		page.Filter = []string{"Cari pekerja", "Rentang tanggal"}
		page.Kolom = []string{"Pekerja", "Tanggal ambil", "Jumlah", "Sudah dipotong", "Saldo"}
		page.Pesan = "Saldo kasbon berkurang setelah pembayaran dicatat."
		query = `SELECT p.nama,DATE_FORMAT(k.tanggal,'%d %b %Y'),CONCAT('Rp',FORMAT(k.jumlah,0,'de_DE')),CONCAT('Rp',FORMAT(COALESCE(paid.total,0),0,'de_DE')),CONCAT('Rp',FORMAT(k.jumlah-COALESCE(paid.total,0),0,'de_DE')) FROM kasbon k JOIN pekerja p ON p.id_pekerja=k.id_pekerja LEFT JOIN (SELECT pk.id_kasbon,SUM(pk.jumlah) AS total FROM potongan_kasbon pk JOIN gaji g ON g.id_gaji=pk.id_gaji WHERE g.status_bayar='dibayar' GROUP BY pk.id_kasbon) paid ON paid.id_kasbon=k.id_kasbon ORDER BY k.tanggal DESC`
	case "/periode":
		page.Judul = "Penggajian"
		page.Subjudul = "Periode dan rincian penggajian"
		page.Aksi = "Hitung Gaji"
		page.Filter = []string{"Jenis periode", "Periode", "Jenis pekerja"}
		page.Kolom = []string{"Pekerja", "Jenis", "Pokok", "Lembur", "Potongan", "Koreksi", "Bersih", "Status"}
		page.Metrik = []ModulMetric{{"Gaji bruto", "—"}, {"Potongan kasbon", "—"}, {"Pembayaran bersih", "—"}}
		query = `SELECT p.nama,p.jenis_pekerja,CONCAT('Rp',FORMAT(g.gaji_pokok,0,'de_DE')),CONCAT('Rp',FORMAT(g.total_lembur,0,'de_DE')),CONCAT('Rp',FORMAT(g.potongan_telat+g.total_potongan_kasbon,0,'de_DE')),CONCAT('Rp',FORMAT(g.total_koreksi,0,'de_DE')),CONCAT('Rp',FORMAT(g.gaji_bersih,0,'de_DE')),REPLACE(g.status_bayar,'_',' ') FROM gaji g JOIN pekerja p ON p.id_pekerja=g.id_pekerja JOIN periode_gaji pg ON pg.id_periode=g.id_periode ORDER BY pg.tanggal_awal DESC,p.nama`
	case "/koreksi":
		page.Judul = "Koreksi Gaji"
		page.Subjudul = "Pengajuan dan persetujuan koreksi antarperiode"
		page.Aksi = "Ajukan Koreksi"
		page.Filter = []string{"Periode asal", "Periode tujuan", "Pekerja", "Status"}
		page.Kolom = []string{"Pekerja", "Periode asal", "Periode tujuan", "Selisih", "Alasan", "Status"}
		page.Pesan = "Koreksi yang diajukan belum masuk perhitungan gaji tujuan."
		query = `SELECT p.nama,DATE_FORMAT(pa.tanggal_awal,'%d %b %Y'),COALESCE(DATE_FORMAT(pt.tanggal_awal,'%d %b %Y'),'—'),CONCAT('Rp',FORMAT(k.nilai_selisih,0,'de_DE')),k.alasan,k.status FROM koreksi_gaji k JOIN gaji ga ON ga.id_gaji=k.id_gaji_asal JOIN pekerja p ON p.id_pekerja=ga.id_pekerja JOIN periode_gaji pa ON pa.id_periode=ga.id_periode LEFT JOIN gaji gt ON gt.id_gaji=k.id_gaji_tujuan LEFT JOIN periode_gaji pt ON pt.id_periode=gt.id_periode ORDER BY k.created_at DESC`
	case "/laporan/rekap":
		page.Judul = "Laporan"
		page.Subjudul = "Rekap biaya gaji dan lembur per periode"
		page.Filter = []string{"Periode", "Jenis pekerja"}
		page.Kolom = []string{"Pekerja", "Pokok", "Lembur", "Terlambat", "Kasbon", "Koreksi", "Bruto", "Bersih", "Status"}
		page.Metrik = []ModulMetric{{"Biaya gaji bruto", "—"}, {"Pembayaran bersih", "—"}}
		query = `SELECT p.nama,CONCAT('Rp',FORMAT(g.gaji_pokok,0,'de_DE')),CONCAT('Rp',FORMAT(g.total_lembur,0,'de_DE')),CONCAT('Rp',FORMAT(g.potongan_telat,0,'de_DE')),CONCAT('Rp',FORMAT(g.total_potongan_kasbon,0,'de_DE')),CONCAT('Rp',FORMAT(g.total_koreksi,0,'de_DE')),CONCAT('Rp',FORMAT(g.gaji_bruto,0,'de_DE')),CONCAT('Rp',FORMAT(g.gaji_bersih,0,'de_DE')),REPLACE(g.status_bayar,'_',' ') FROM gaji g JOIN pekerja p ON p.id_pekerja=g.id_pekerja JOIN periode_gaji pg ON pg.id_periode=g.id_periode ORDER BY pg.tanggal_awal DESC,p.nama`
	default:
		http.NotFound(w, r)
		return
	}
	rows, err := h.DB.QueryContext(r.Context(), query)
	if err != nil {
		http.Error(w, "Data tidak dapat dimuat", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	for rows.Next() {
		values := make([]sql.NullString, len(page.Kolom))
		args := make([]any, len(values))
		for i := range values {
			args[i] = &values[i]
		}
		if err := rows.Scan(args...); err != nil {
			http.Error(w, "Data tidak dapat dibaca", http.StatusInternalServerError)
			return
		}
		row := make([]string, len(values))
		for i, v := range values {
			row[i] = strings.TrimSpace(v.String)
		}
		page.Baris = append(page.Baris, row)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Data tidak dapat dibaca", http.StatusInternalServerError)
		return
	}
	h.View.Tampil(w, "modul.html", page)
}
