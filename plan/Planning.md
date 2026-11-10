# Panduan Pengerjaan Sistem Penggajian Konveksi Mitra Jaya

Panduan ini menerjemahkan rancangan yang dikumpulkan dalam [dokumen persiapan tahap 1](<Project Persiapan Tahap 1 Kelompok 3_nawaitu.docx>) menjadi pembagian pekerjaan dan kontrak integrasi. Dokumen tersebut menjadi acuan utama untuk aturan bisnis, hak akses, kamus data, dan use case. [Sketsa Figma](https://www.figma.com/design/4lAH0BVRuWmHQxhgifYpHu/) menjadi acuan tampilan desktop, tablet, dan mobile.

Proyek masih berupa starter: autentikasi, kerangka navigasi, dan pembacaan awal beberapa halaman sudah tersedia. Formulir modul, transaksi, perhitungan, dan laporan belum lengkap. Tombol yang belum memiliki backend tidak boleh dianggap sebagai fitur yang sudah selesai.

## Pembagian pekerjaan lima anggota

| Anggota | Modul dan hasil yang dikerjakan | Tabel utama |
| --- | --- | --- |
| Muhammad Naufal A. (2404130057) | Konfigurasi, login dan sesi, hak akses Admin/Pemilik, layout bersama, dashboard, integrasi route dan skema | `pengguna`, `sesi` |
| Cahya Din Ramadhan (2404130018) | Data pekerja, identitas dan masa kerja, riwayat upah, serta pencarian tarif berdasarkan tanggal | `pekerja`, `riwayat_upah` |
| Muhammad Falih A. (2404130054) | Shift, penempatan jadwal, absensi, pemeriksaan data belum lengkap, keterlambatan dan durasi lembur | `shift`, `jadwal_shift`, `absensi` |
| Daffa Azmi Ikhtiar F. (2404130059) | Periode mingguan/bulanan, rincian dan perhitungan gaji, pembayaran, penutupan periode dan penguncian | `periode_gaji`, `gaji`, `detail_gaji` |
| Adel Bin Faris Abdat (2404130169) | Kasbon dan alokasi potongannya, koreksi, slip individual, serta rekap gaji dan lembur | `kasbon`, `potongan_kasbon`, `koreksi_gaji` |

Setiap anggota mengembangkan tampilan modulnya dengan HTML, CSS, dan JavaScript vanilla serta backend Go dan MySQL. Koordinasikan perubahan pada file bersama (`cmd/web/main.go`, `templates/layout.html`, `templates/sidebar.html`, `internal/handler/modul.go`, dan `db/schema.sql`) agar tidak saling menimpa. Halaman starter saat ini berbagi `templates/modul.html`; pisahkan template dan handler per modul ketika pekerjaan masing-masing mulai diimplementasikan. Jangan menganggap query atau form starter sebagai implementasi akhir.

## Hak akses dan halaman

Hanya ada dua peran aplikasi. **Admin Penggajian** mengelola pekerja, upah, jadwal, absensi, kasbon, penghitungan, pembayaran, dan pengajuan koreksi; Admin dapat mencetak slip dan melihat rekap. **Pemilik** membaca data operasional dan hasil penggajian, menyetujui koreksi, serta menutup periode setelah seluruh gaji dibayar. Pemilik melihat rekap. Pekerja tidak memiliki akun; Supervisor/Petugas Absensi bukan peran terpisah dalam rancangan.

Sembilan halaman adalah Login, Dashboard, Data Pekerja, Shift dan Jadwal, Absensi, Kasbon, Penggajian, Koreksi Gaji, dan Laporan. Pada 390 px dan 768 px, ikuti frame responsif Figma: navigasi berbentuk drawer, filter membungkus, dan tabel dapat digulir dengan identitas pekerja tetap terlihat. Detail gaji menempatkan gaji bersih dan status di dekat bagian atas. Kontrol UI dan validasi di server harus sama-sama menerapkan hak akses; menyembunyikan tombol saja tidak cukup.

## Alur dan aturan bisnis

1. **Pekerja dan upah (AB 01–02).** Simpan pekerja harian atau bulanan dan riwayat upah dengan tanggal berlaku unik per pekerja. Tarif pada suatu tanggal adalah tarif terakhir yang mulai berlaku pada atau sebelum tanggal tersebut. Pekerja yang berhenti tetap disimpan; tanggal masuk dan keluar termasuk masa aktif.
2. **Jadwal dan absensi (AB 03).** Shift Pagi 07.00–15.00 dan Siang 15.00–23.00. Maksimal satu jadwal per pekerja per tanggal, dalam masa aktif. Jadwal tanpa catatan tidak otomatis tidak hadir. Hadir terkonfirmasi membutuhkan jam masuk dan keluar; tidak hadir tidak memiliki jam. Catatan yang belum lengkap perlu diperiksa. Jam keluar tidak boleh lebih awal daripada jam masuk. Kerja lintas tengah malam tidak termasuk cakupan awal.
3. **Periode (AB 01, 10).** Pekerja harian menggunakan periode Senin–Sabtu dan dibayar Sabtu; pekerja bulanan menggunakan bulan kalender dengan tanggal bayar yang ditentukan saat membuat periode. Periode sejenis tidak boleh bertumpang tindih. Penutupan mingguan hanya mengunci absensi pekerja harian yang terkait.
4. **Tarif dan pokok (AB 04–05).** Tarif per jam adalah upah harian ÷ 8 atau gaji bulanan ÷ 173. Pokok harian menjumlahkan tarif yang berlaku pada setiap tanggal hadir terkonfirmasi. Pokok bulanan dijumlahkan per tanggal aktif: tarif bulanan yang berlaku pada tanggal itu ÷ jumlah hari kalender dalam bulan. Ini menangani prorata dan perubahan tarif di tengah bulan. Tidak ada potongan ketidakhadiran bulanan tambahan di luar ketentuan dokumen.
5. **Terlambat dan lembur (AB 06–07).** Menit terlambat adalah maksimum nol dan selisih jam masuk dengan awal shift. Hanya keterlambatan **lebih dari** 15 menit dipotong satu jam upah per kejadian; tepat 15 menit tidak dipotong. Dengan H = menit lembur ÷ 60, lembur = tarif per jam × (1,5 × min(H, 1) + 2 × max(H − 1, 0)). Hitungan jam pertama dimulai ulang setiap tanggal; datang lebih awal bukan lembur.
6. **Kasbon (AB 08).** Batas potongan adalah 30% dari bruto sebelum potongan dan koreksi, dibulatkan ke bawah ke rupiah. Jumlah yang dipotong tidak melebihi saldo kasbon atau sisa gaji tersedia. Gunakan kasbon tertua yang tanggalnya tidak melewati akhir periode. Saldo adalah pengambilan dikurangi potongan pada gaji yang sudah dibayar; draf tidak mengurangi saldo. Jika nilai sebelum kasbon negatif, tahan pembayaran untuk pemeriksaan.
7. **Bruto, bersih, dan pembulatan (AB 09).** Bruto = pokok + lembur. Bersih = bruto − keterlambatan − kasbon + koreksi bersih. Simpan uang dalam desimal; bulatkan gaji akhir ke rupiah terdekat, pecahan 0,5 naik. Hindari mengandalkan ketepatan `float64` untuk nominal uang; gunakan representasi desimal atau bilangan bulat dengan aturan pembulatan yang diuji.
8. **Pembayaran dan tutup (AB 10).** Sebelum pembayaran, periksa ulang saldo kasbon, batas potongan, dan koreksi. Satu gaji dibayar penuh satu kali; simpan waktu bayar. Draf belum dibayar pada periode terbuka dapat dihitung ulang tanpa menggandakan rincian atau potongan. Setelah seluruh gaji dibayar, hanya Pemilik dapat menutup periode; absensi dan hasil terkait dikunci. Gaji yang dibayar tidak dihitung ulang.
9. **Koreksi (AB 11).** Admin mengajukan koreksi atas gaji yang telah dibayar, dengan alasan dan selisih bertanda. Pemilik menyetujui. Koreksi diterapkan sekali pada gaji pekerja yang sama di periode berikutnya, dengan memperhitungkan koreksi sebelumnya untuk kesalahan yang sama. Pekerja yang sudah berhenti tetap dapat menerima penyelesaian koreksi tanpa pokok baru. Pembayaran, potongan dan penerapan koreksi harus atomik dalam satu transaksi.
10. **Laporan.** Slip individual dan rekap per periode memisahkan pokok, lembur, keterlambatan, kasbon, koreksi, bruto, dan pembayaran bersih. Rekap dapat disaring berdasarkan periode dan jenis pekerja.

## Struktur data dan kontrak integrasi

`db/schema.sql` menyediakan 12 entitas rancangan (`pengguna`, `pekerja`, `riwayat_upah`, `shift`, `jadwal_shift`, `absensi`, `periode_gaji`, `gaji`, `detail_gaji`, `kasbon`, `potongan_kasbon`, dan `koreksi_gaji`) dengan penamaan kolom yang selaras langsung dengan Subbab 3.1–3.3 Kamus Data Ringkas pada dokumen `Project Persiapan Tahap 1 Kelompok 3_nawaitu.docx`, ditambah `sesi` dan tabel contoh `produk` untuk aplikasi starter. `sesi` dan `produk` bukan entitas tambahan pada ERD rancangan.

`detail_gaji.id_absensi` boleh kosong untuk rincian bulanan tanpa absensi dan maksimal satu rincian untuk setiap pasangan gaji/tanggal; satu absensi tidak boleh digunakan dua kali. Gaji penyelesaian koreksi dapat tidak memiliki rincian kerja baru. Kolom `perlu_diperiksa` pada status absensi adalah kondisi sementara untuk catatan belum lengkap, bukan hadir terkonfirmasi. `id_gaji_tujuan` dan persetujuan koreksi boleh kosong saat masih dalam proses. Saldo kasbon dihitung dari transaksi yang sudah dibayar, bukan dikurangi pada draf.

Validasi yang memerlukan tabel atau transaksi lain—termasuk periode tidak tumpang tindih, masa aktif pekerja, pemilik absensi/upah/kasbon/koreksi sama dengan penerima gaji, tarif efektif, dan penguncian setelah pembayaran—harus ditegakkan pada service dalam transaksi. Skema saja tidak menjamin aturan tersebut. Perubahan skema untuk database yang sudah ada membutuhkan migrasi tersendiri; **jangan menjalankan `schema.sql` terhadap database berisi data**, karena file itu membuat ulang tabel.

## Urutan integrasi dan pengujian

1. Naufal menyiapkan konfigurasi, autentikasi, layout dan akses dua peran. Cahya menyelesaikan pekerja/riwayat upah; Falih mengerjakan jadwal dan absensi.
2. Daffa mengintegrasikan data pekerja/upah dan absensi ke perhitungan. Adel menyiapkan kasbon dan koreksi; Daffa dan Adel menyepakati kontrak transaksi untuk draf, pembayaran, dan penerapan koreksi.
3. Integrasikan laporan, uji skenario Admin dan Pemilik, serta bandingkan desktop, tablet, dan mobile terhadap Figma. Jalankan `go test ./...` dan uji integrasi database pada database uji yang terpisah.

Kasus minimum: terlambat 15 dan 16 menit; lembur pecahan dan dua jam; perubahan tarif di tengah periode; prorata 16/31; kasbon lebih besar dari batas/sisa dan setelah akhir periode; nilai sebelum kasbon negatif; hitung ulang draf; pencegahan bayar ganda; penutupan saat ada gaji belum dibayar; koreksi satu kali untuk pekerja yang sama termasuk yang sudah berhenti; serta penolakan aksi Admin/Pemilik yang tidak berwenang. Data contoh dan perhitungan Ayu/Bima di dokumen Word dapat dipakai sebagai pembanding, bukan data produksi.
