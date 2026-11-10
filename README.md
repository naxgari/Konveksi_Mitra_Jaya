# **Pembagian Tugas 5 Anggota (Full-Stack per Modul)** 

Agar adil dan setiap orang memahami alur Go dari database sampai ke tampilan web, bagi tugas secara vertikal per modul fitur (setiap orang menggarap `model` -> `repository` -> `service` -> `handler` -> `template HTML` untuk modulnya masing-masing): 

## 1. Muhammad Naufal A. (Ketua) — Modul Auth, Middleware, Core Setup & Integrasi 

Karena materi kuliah hari ini fokus penuh pada Authentication dan kamu adalah ketua yang harus mengawal integrasi kode, kamu memegang pondasi keamanan dan kerangka utama aplikasi: 

   - Database & Konfigurasi: Menyusun file `db/schema.sql` lengkap sesuai ERD, mengatur koneksi di `config.go`, `.env` (`COOKIE_SECURE`), dan merakit routing utama (pemisahan Public vs Private `ServeMux`) di `cmd/web/main.go`. 

   - Fitur Autentikasi (10 Langkah Dosen): Menggarap tabel `PENGGUNA` & `SESI` (`sessions`), fitur Login, Register (validasi password kuat + hashing `bcrypt` + tangkap error MySQL `1062`), dan Logout. 

   - Middleware & Role Access: Membuat middleware `RequireLogin` (baca token dari Cookie & simpan data user ke `context.WithValue`) serta pembatasan akses untuk dua peran sesuai dokumen: Admin Penggajian dan Pemilik.

   - Template Utama: Merapikan `layout.html`, `sidebar.html`, halaman `/login`, dan `/register`. 

## 2. Cahya Din Ramadhan — Modul Master Pekerja & Riwayat Upah 

Bertanggung jawab atas data induk pekerja dan pelacakan perubahan gaji mereka (Aturan Bisnis AB 02 ): 

- Tabel yang Diurus: `PEKERJA` dan `RIWAYAT_UPAH`. 

- Fitur Backend (`model`, `repo`, `service`, `handler`): - CRUD data pekerja (nama, jenis pekerja: `harian`/`bulanan`, jabatan, kontak, tanggal masuk, tanggal keluar, status aktif). Pekerja yang berhenti tidak boleh dihapus (soft deactive), hanya di-set status/tanggal keluarnya. - Pencatatan dan penelusuran Riwayat Upah beserta `berlaku_mulai`. - Membuat fungsi helper di repository/service untuk mengambil tarif upah yang berlaku pada tanggal tertentu (fungsi ini nanti akan dipakai oleh pembuat Modul Perhitungan Gaji). 

- Template HTML: Halaman daftar pekerja, form tambah/edit pekerja, dan halaman detail pekerja yang menampilkan tabel riwayat perubahan upahnya. 

## 3. Muhammad Falih A — Modul Jadwal Shift & Pencatatan Absensi 

Bertanggung jawab atas operasional harian yang dikelola Admin dan dapat dibaca oleh Pemilik (Aturan Bisnis AB 03, AB 06, AB 07).

   - Tabel yang Diurus: `SHIFT`, `JADWAL_SHIFT`, dan `ABSENSI`. 

   - Fitur Backend (`model`, `repo`, `service`, `handler`): - Pengelolaan master Shift (Pagi `07:00-15:00` dan Siang `15:00-23:00`) dan penugasan Jadwal Shift per pekerja per tanggal (validasi maksimal 1 shift per pekerja per tanggal). - Input jam masuk & jam keluar pekerja (validasi: jam keluar tidak boleh mendahului jam masuk, serta tidak bisa diedit jika periode gaji terkait sudah berstatus Ditutup ). - Logika perhitungan otomatis menit terlambat ( _max_ ( 0 _, jammasukaktual_ − _awalshift_ )) dan jam lembur saat absensi disimpan, serta penandaan (flagging) jadwal yang belum lengkap jam absensinya untuk diperiksa. 

   - Template HTML: Halaman kelola jadwal shift, form input absensi harian, dan daftar rekap kehadiran/pemeriksaan absensi. 

## 4. Daffa Azmi Ikhtiar F. — Modul Periode & Engine Perhitungan Gaji 

Ini adalah modul inti (core engine) yang mengolah data dari modul Pekerja dan Absensi menjadi draf gaji (Aturan Bisnis AB 01, AB 04, AB 05a, AB 05b, AB 09a, AB 09b, AB 10 ): 

- Tabel yang Diurus: `PERIODE_GAJI`, `GAJI`, dan `DETAIL_GAJI`. 

- Fitur Backend (`model`, `repo`, `service`, `handler`): - Pembuatan periode gaji (mingguan Senin–Sabtu untuk harian, bulanan untuk pekerja bulanan). - Engine Hitung Gaji: Menghitung upah per jam (harian _÷_ 8, bulanan _÷_ 173), gaji pokok harian, gaji proporsional/prorata bulanan (berdasarkan hari kalender aktif), upah lembur progresif ( 1.5 _×_ jam pertama, 2 _×_ jam berikutnya), dan potongan keterlambatan ( >15 menit potong 1 jam upah). - Fitur Pembayaran Gaji (mencatat waktu bayar & mencegah bayar dua kali) dan Penutupan Periode (mengunci periode dan absensi setelah semua gaji pada periode tersebut lunas dibayar). 

- Template HTML: Halaman daftar periode gaji, halaman rincian perhitungan gaji per periode, dan tampilan draf rincian gaji per tanggal (`DETAIL_GAJI`). 

## 5. Adel Bin Faris Abdat — Modul Kasbon, Koreksi Antarperiode & Laporan/Slip 

Bertanggung jawab atas pemotongan pinjaman pekerja, penanganan selisih/kesalahan setelah tutup periode, dan output laporan akhir (Aturan Bisnis AB 08, AB 11 ): 

- Tabel yang Diurus: `KASBON`, `POTONGAN_KASBON`, dan `KOREKSI_GAJI`. 

- Fitur Backend (`model`, `repo`, `service`, `handler`): - Pencatatan pengajuan kasbon pekerja, pelacakan saldo sisa, dan logika pemotongan kasbon otomatis saat hitung gaji (maksimal 30% dari gaji bruto dibulatkan ke bawah, memotong kasbon tertua lebih dulu yang tanggalnya tidak melewati akhir periode). - Fitur pencatatan Koreksi Gaji (menghubungkan gaji asal yang sudah dibayar ke gaji tujuan di periode berikutnya beserta alasan, nilai selisih, dan persetujuan pemilik). - Pembuatan halaman Slip Gaji Individual (siap cetak/print dari browser) dan Laporan Rekap Biaya Gaji & Lembur per Periode (dengan filter periode dan jenis pekerja). 

- Template HTML: Halaman kelola kasbon, form & daftar koreksi gaji, halaman cetak slip gaji, dan dashboard rekap laporan. 

# **Tabel Ringkasan Pembagian Tugas & Dokumentasi Laporan** 

Karena tadi di akhir kelas ada yang bertanya soal laporan dan Pak Anggyi membolehkan penambahan Flowchart / Sequence Diagram , sekalian bagi tugas penulisan dokumen laporan akhirnya agar tidak menumpuk di kamu sendiri: 

|Nama<br>Anggot<br>a|Fokus Modul Coding (Go + MySQL<br>+ HTML)|Tugas Laporan & Presentasi<br>(Pertemuan 15)|
|---|---|---|
|Naufal<br>(Ketua)|Setup Project, Schema SQL, Auth<br>(`bcrypt` + Session Cookie),<br>Middleware, Routing `main.go`|Flowchart/Sequence Diagram<br>Auth, penggabungan dokumen<br>laporan & penyusunan slide<br>presentasi|
|Cahya|CRUD Pekerja (`harian`/`bulanan`)<br>& Riwayat Upah (`PEKERJA`,<br>`RIWAYAT_UPAH`)|<br>Flowchart/Sequence Diagram<br>Modul Pekerja & Riwayat Upah<br>+ Screenshot hasil|
|Falih|Master Shift, Jadwal Shift, Input &<br>Validasi Absensi (`SHIFT`,<br>`JADWAL_SHIFT`, `ABSENSI`)|Flowchart/Sequence Diagram<br>Modul Shift & Absensi +<br>Screenshot hasil|
|Daffa|Periode Gaji, Engine Hitung|Flowchart/Sequence Diagram|



||Gaji/Lembur/Prorata, Tutup Period<br>(`PERIODE_GAJI`, `GAJI`,<br>`DETAIL_GAJI`)|e<br>Perhitungan Gaji & Penutupan<br>Periode + Screenshot hasil|
|---|---|---|
|Adel|Kasbon (Potongan 30%), Koreksi<br>Gaji, Slip Gaji & Rekap Laporan<br>(`KASBON`,<br>`POTONGAN_KASBON`,<br>`KOREKSI_GAJI`)|Flowchart/Sequence Diagram<br>Kasbon & Koreksi + Uji kasus<br>perhitungan (testing)|



# **Urutan Pengerjaan (Timeline)** 

1. Tahap 1 (Minggu Ini): Kamu menyelesaikan Setup Repo, `db/schema.sql`, dan Modul Auth berbasis `go-produk`. Begitu selesai, push ke GitHub agar Cahya, Falih, Daffa, dan Adel bisa langsung `git clone` dan mulai ngoding di struktur yang sudah jalan. 

2. Tahap 2: Cahya (Pekerja & Upah) dan Falih (Shift & Absensi) menyelesaikan modul mereka lebih dulu karena modul Daffa (Hitung Gaji) membutuhkan data dari tabel mereka. Sambil menunggu, Daffa dan Adel sudah bisa membuat CRUD dasar untuk `PERIODE_GAJI` dan `KASBON`. 

3. Tahap 3: Integrasi perhitungan gaji ( Daffa ) dengan potongan kasbon & koreksi ( Adel ), lalu pengujian keseluruhan alur dari tahap 1 sampai 6 sesuai tabel Alur Kerja Sistem di Bab 1.3 dokumen kalian. 

