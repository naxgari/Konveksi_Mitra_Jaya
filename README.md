# Sistem Penggajian Pekerja Harian & Shift — Konveksi Mitra Jaya (Kelompok 3)

Aplikasi web full-stack berbasis **Go (`net/http`)**, **MySQL**, dan **HTML/CSS/JavaScript vanilla** untuk mengotomatiskan pengelolaan pekerja, riwayat upah, jadwal shift, absensi, kasbon, perhitungan gaji (harian & bulanan prorata, lembur progresif, potongan keterlambatan), penutupan periode, koreksi antarperiode, slip gaji, serta rekap laporan biaya gaji pada **Konveksi Mitra Jaya**.

---

## Pembagian Tugas 5 Anggota (Full-Stack Vertikal per Modul Kode)

Setiap anggota bertanggung jawab secara penuh dan adil atas modul kode programnya masing-masing secara vertikal (**`model` → `repository` → `service` → `handler` → `template HTML` → `test`**) sehingga tidak ada peran "merapikan secara global" atau *one-man-show*.

### Tabel Ringkasan Pembagian Job Desc

| No | Nama Anggota | NIM | Modul & Ruang Lingkup Kode | Tabel Database | File Kode Utama |
|---|---|---|---|---|---|
| 1 | **Muhammad Naufal A.** | `2404130057` | Autentikasi (`bcrypt` + Sesi Cookie), Middleware Hak Akses 2 Peran, Konfigurasi DB, Routing & Dashboard | `pengguna`, `sesi` | `cmd/web/main.go`, `internal/config/`, `internal/middleware/`, `internal/*/pengguna.go`, `internal/service/auth.go`, `internal/handler/{auth,halaman,render,helper}.go`, `templates/{layout,sidebar,home,profil,auth/*}.html` |
| 2 | **Cahya Din Ramadhan** | `2404130018` | Master Pekerja (`harian`/`bulanan`), Soft-Deactive, Penetapan Upah Awal & Riwayat Perubahan Upah (`AB 01–02`) | `pekerja`, `riwayat_upah` | `internal/{model,repository,service,handler}/{pekerja,riwayat_upah}.go`, `templates/pekerja/*.html` |
| 3 | **Muhammad Falih A.** | `2404130054` | Master Shift, Penjadwalan Shift (Tunggal & Rentang), Pencatatan & Pemeriksaan Absensi (`AB 03, AB 06–07`) | `shift`, `jadwal_shift`, `absensi` | `internal/{model,repository,service,handler}/absensi.go`, `internal/*/absensi*_test.go`, `templates/absensi/*.html` |
| 4 | **Daffa Azmi Ikhtiar F.** | `2404130059` | Periode Gaji, Engine Hitung Gaji/Lembur/Prorata (`big.Rat`), Pembayaran Atomik & Penutupan Periode (`AB 01, AB 04–07, AB 09–10`) | `periode_gaji`, `gaji`, `detail_gaji` | `internal/{model,repository,service}/{periode_gaji,penggajian}.go`, `internal/service/{perhitungan_gaji,draf_gaji}.go`, `internal/handler/penggajian.go`, `templates/gaji/*.html` |
| 5 | **Adel Bin Faris Abdat** | `2404130169` | Kasbon Pekerja (FIFO Maks. 30%), Koreksi Antarperiode, Cetak Slip Gaji & Rekap Laporan (`AB 08, AB 11`) | `kasbon`, `potongan_kasbon`, `koreksi_gaji` | `internal/{model,repository,service,handler}/kasbon.go`, `internal/service/kasbon_test.go`, `templates/kasbon/*.html` |

---

### Rincian Tanggung Jawab Kode per Anggota

#### 1. Muhammad Naufal A. (`2404130057`) — Modul Autentikasi, Middleware, Konfigurasi & Dashboard
- **Konfigurasi & Koneksi Database:** Menyusun `db/schema.sql`, konfigurasi `.env` & `internal/config/config.go`, koneksi MySQL (`loc=Local`, connection pool) serta perakitan routing Public vs Private `ServeMux` + `CrossOriginProtection` di `cmd/web/main.go`.
- **Autentikasi & Sesi (`pengguna`, `sesi`):** Implementasi Login, Register (validasi panjang input, password kuat + hashing `bcrypt`, penanganan error duplikat MySQL `1062`), Logout, auto-seeder akun default (`admin` & `pemilik`), dan pengelolaan sesi cookie (`HttpOnly`, `SameSite`).
- **Middleware & Hak Akses 2 Peran:** Implementasi `RequireLogin` (injeksi pengguna ke `context` dan `UserResponseWriter`), `RequireRole` untuk pemisahan akses **Admin Penggajian** dan **Pemilik**, serta middleware logging request.
- **Renderer, Layout & Dashboard:** Implementasi `Renderer` (`Tampil` / `TampilStatus`, helper `rupiah_upah`, `tgl_indo`), kerangka tampilan bersama (`layout.html`, `sidebar.html`, `profil.html`), serta query & tampilan KPI real-time Dashboard (`home.html`).

#### 2. Cahya Din Ramadhan (`2404130018`) — Modul Master Pekerja & Riwayat Upah (`AB 01–02`)
- **Master Pekerja (`pekerja`):** CRUD data pekerja (`nama`, `jenis_pekerja`: `harian`/`bulanan`, `jabatan`, `kontak`, `tanggal_masuk`, `tanggal_keluar`) termasuk penetapan **Upah Awal** atomik dalam 1 transaksi saat pendaftaran pekerja baru, serta mekanisme *soft-deactive* (mengisi `tanggal_keluar` tanpa menghapus baris pekerja agar riwayat gaji tetap utuh).
- **Riwayat Perubahan Upah (`riwayat_upah`):** Pencatatan perubahan tarif upah beserta `berlaku_mulai` (`UNIQUE(id_pekerja, berlaku_mulai)`), validasi rentang masa aktif pekerja, validasi nominal `DECIMAL(15,2)` positif, serta fungsi `TarifBerlaku` untuk mengambil tarif aktif pada tanggal tertentu.
- **Tampilan Modul (`templates/pekerja/`):** Halaman daftar pekerja responsif, form tambah/edit pekerja, detail pekerja, serta daftar & form tambah riwayat upah (dengan pembatasan tombol aksi khusus Admin).

#### 3. Muhammad Falih A. (`2404130054`) — Modul Master Shift, Jadwal Shift & Absensi (`AB 03, AB 06–07`)
- **Master Shift & Penjadwalan (`shift`, `jadwal_shift`):** Pengelolaan master shift (Pagi `07:00–15:00`, Siang `15:00–23:00`), penempatan jadwal tunggal maupun rentang tanggal (`JadwalBuatRentang`, opsi lewati Minggu) dengan validasi maksimal 1 shift per pekerja per tanggal dalam masa aktif.
- **Pencatatan & Pemeriksaan Absensi (`absensi`):** Input absensi manual maupun konfirmasi cepat kehadiran normal (`Hadir Normal` satuan & massal), validasi pasangan jam masuk/keluar (jam keluar ≥ jam masuk, tidak lintas tanggal), penandaan status `perlu_diperiksa`, perhitungan otomatis `menit_terlambat` & `menit_lembur`, serta penguncian perubahan apabila periode gaji terkait sudah ditutup atau gaji sudah dibayar.
- **Tampilan & Pengujian (`templates/absensi/`, `absensi*_test.go`):** Halaman kelola shift, jadwal, form absensi, dan pemeriksaan absensi beserta unit test & integration test MySQL.

#### 4. Daffa Azmi Ikhtiar F. (`2404130059`) — Modul Periode & Engine Perhitungan Gaji (`AB 01, AB 04–07, AB 09–10`)
- **Manajemen Periode Gaji (`periode_gaji`):** Pembuatan periode mingguan (Senin–Sabtu, bayar Sabtu) dan bulanan (bulan kalender) dengan validasi anti tumpang tindih pada transaksi `SERIALIZABLE`.
- **Engine Hitung Gaji (`gaji`, `detail_gaji`):** Perhitungan presisi berbasis `big.Rat` (tanpa `float64`) meliputi tarif per jam (`harian ÷ 8`, `bulanan ÷ 173`), gaji pokok harian & prorata bulanan per hari aktif (mendukung perubahan tarif di tengah bulan), upah lembur progresif (`1,5×` jam pertama, `2×` jam berikutnya), potongan keterlambatan (`>15` menit potong 1 jam), integrasi alokasi kasbon & koreksi, serta pembulatan gaji akhir (`0,5` naik).
- **Pembayaran Atomik & Penutupan Periode:** Hitung draf satuan & massal (`HitungSemuaDraf`), pembayaran gaji atomik anti-bayar ganda (mengunci rincian, memfinalisasi potongan kasbon & menerapkan koreksi), serta penutupan periode khusus Pemilik setelah seluruh pekerja pada periode tersebut dibayar.
- **Tampilan & Pengujian (`templates/gaji/`, `penggajian_test.go`, `hitung_draf_test.go`, `penutupan_periode_test.go`):** Halaman daftar/form/detail periode dan detail rincian gaji per tanggal beserta rangkaian unit test & integration test MySQL.

#### 5. Adel Bin Faris Abdat (`2404130169`) — Modul Kasbon, Koreksi Antarperiode, Slip & Laporan (`AB 08, AB 11`)
- **Manajemen Kasbon (`kasbon`, `potongan_kasbon`):** Pencatatan pengajuan kasbon dalam masa aktif pekerja, pelacakan saldo sisa (hanya berkurang setelah gaji berstatus `dibayar`; draf tidak mengurangi saldo), riwayat pemotongan per kasbon, serta aturan pemotongan FIFO maksimal 30% gaji bruto (dibulatkan ke bawah ke rupiah) dengan penahanan pembayaran bila nilai sebelum kasbon negatif.
- **Koreksi Gaji Antarperiode (`koreksi_gaji`):** Pengajuan koreksi oleh Admin atas gaji yang sudah dibayar (selisih positif/negatif + alasan), persetujuan oleh Pemilik, dan penerapan satu kali pada gaji pekerja yang sama di periode berikutnya (termasuk pekerja yang sudah berhenti tanpa pokok baru).
- **Slip Gaji & Rekap Laporan (`templates/kasbon/`):** Halaman Slip Gaji individual siap cetak (`@media print`) dan Laporan Rekap Biaya Gaji & Lembur per periode dengan filter periode & jenis pekerja serta baris total seluruh komponen.

---

## Prosedur & Aturan Presentasi Project Kelompok

- **Tanpa File Presentasi:** Tidak perlu membuat slide/file presentasi.
- **Batas Waktu Ketat:** Maksimal **15 menit per kelompok** termasuk persiapan. Kelompok wajib langsung maju dan memulai saat gilirannya tiba serta selesai tepat waktu.
- **Susunan Presentasi:**
  1. **Demo Program Langsung (±3 menit):** Dilakukan oleh **1 perwakilan kelompok** (bukan bergantian) untuk mendemokan alur aplikasi berjalan secara langsung **tanpa menjelaskan logic isi kode program**:
     - Login **Admin** → Dashboard → Tambah pekerja + upah awal → Jadwal shift rentang + absensi cepat (1 telat/lembur) → Catat kasbon → Buat periode & hitung gaji + bayar gaji.
     - Login **Pemilik** → Cek hak akses baca saja → Tutup periode → Setujui koreksi gaji.
     - Tampilkan **Slip Gaji** (siap cetak) & **Laporan Rekap** (filter periode/jenis).
  2. **Sesi Tanya Jawab (±10–12 menit):** Pertanyaan langsung dari dosen kepada **masing-masing anggota** sesuai job desc kode programnya pada tabel di atas.

---

## Cara Menjalankan Aplikasi & Pengujian

1. Pastikan **MySQL / MariaDB** berjalan di `127.0.0.1:3306` dan database `konveksi_mitra_jaya` sudah diimpor dari `db/schema.sql`.
2. Sesuaikan konfigurasi pada file `.env`:
   ```env
   ADDR=:8080
   DB_DSN=root:@tcp(127.0.0.1:3306)/konveksi_mitra_jaya?parseTime=true&charset=utf8mb4&loc=Local
   COOKIE_SECURE=false
   ```
3. Jalankan server web:
   ```powershell
   go run ./cmd/web
   ```
4. Buka `http://localhost:8080` di browser. Akun default yang otomatis dibuat saat tabel `pengguna` kosong:
   - **Admin Penggajian:** `admin` / `password123`
   - **Pemilik Konveksi:** `pemilik` / `password123`
5. Menjalankan seluruh unit test (dan integration test MySQL dengan database uji terpisah `konveksi_test`):
   ```powershell
   go test -v ./...
   ```
