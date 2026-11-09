# **Topik Project Akhir Web Programming** 

## **Ketentuan Umum** 

- Project dikerjakan selama 4 minggu atau sampai akhir semester ini. Setiap kelompok mengerjakan satu topik. 

- Dibuat menggunakan Go dan database relasional (MySQL). Tampilan web sederhana sudah cukup. 

- Setiap anggota kelompok harus memahami project dan bagian aplikasi yang menjadi tanggung jawabnya (akan dites saat presentasi project) 


## **Topik Persoalan — Sistem Penggajian Pekerja Harian dan Shift** 

Konveksi Mitra Jaya mempekerjakan sekitar 40 orang yang terdiri dari pekerja harian dan pekerja bulanan. Gaji saat ini dihitung secara manual dari kartu absensi, sehingga sering terjadi kesalahan dalam perhitungan lembur dan potongan. 

Pekerja harian dibayar berdasarkan jumlah hari hadir, dengan periode gaji mingguan (Senin sampai Sabtu) yang dibayarkan setiap hari Sabtu. Pekerja bulanan menerima gaji tetap per bulan. Pekerja bulanan yang mulai bekerja atau berhenti di tengah bulan dibayar secara proporsional. Besaran upah setiap pekerja dapat berubah, misalnya karena kenaikan UMK atau kenaikan jabatan, dan perubahan tersebut berlaku mulai tanggal tertentu. 

Konveksi menerapkan dua shift, yaitu shift pagi pukul 07.00–15.00 dan shift siang pukul 15.00–23.00. Setiap pekerja mencatat jam masuk dan jam keluar. Upah per jam dihitung dari upah harian dibagi 8 untuk pekerja harian, dan dari gaji bulanan dibagi 173 untuk pekerja bulanan. Pekerja yang terlambat lebih dari 15 menit dipotong upah satu jam. 

Pekerja yang bekerja melewati jam shift-nya dihitung lembur. Upah lembur jam pertama sebesar 1,5 kali upah per jam, dan jam-jam berikutnya sebesar 2 kali upah per jam. 

Pekerja dapat mengambil kasbon yang dipotong dari gaji. Potongan kasbon dalam satu periode tidak boleh melebihi 30% dari gaji periode tersebut. Sisa kasbon dipotong pada periode berikutnya. 

Setelah gaji suatu periode diproses dan dibayarkan, periode tersebut ditutup dan data absensinya tidak boleh diubah. Kesalahan yang ditemukan setelah periode ditutup dikoreksi pada periode berikutnya. Pemilik membutuhkan slip gaji setiap pekerja serta rekap biaya gaji dan lembur per periode. 

### **Lingkup aplikasi:** 

1. Pengelolaan data pekerja dan riwayat upah. 

2. Pengelolaan jadwal shift. 

3. Pencatatan absensi (jam masuk dan jam keluar). 

4. Perhitungan gaji per periode, termasuk lembur, potongan keterlambatan, potongan kasbon, dan gaji proporsional. 

5. Penutupan periode gaji dan koreksi. 

6. Slip gaji dan rekap gaji per periode. 