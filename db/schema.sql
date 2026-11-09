-- =============================================================================
-- SCHEMA DATABASE: Sistem Penggajian Pekerja Harian dan Shift
-- Studi Kasus: Konveksi Mitra Jaya (Kelompok 3)
-- Engine: MySQL / MariaDB (InnoDB, utf8mb4)
-- Diselaraskan dengan:
--   1. plan/Topik Project Akhir Web Programming.md
--   2. plan/Project_ Persiapan Tahap 1_Kelompok 3 (1).md (Versi Revisi)
--   3. plan/Planning.md & README.md
-- =============================================================================

SET FOREIGN_KEY_CHECKS = 0;

DROP TABLE IF EXISTS koreksi_gaji;
DROP TABLE IF EXISTS potongan_kasbon;
DROP TABLE IF EXISTS kasbon;
DROP TABLE IF EXISTS detail_gaji;
DROP TABLE IF EXISTS gaji;
DROP TABLE IF EXISTS periode_gaji;
DROP TABLE IF EXISTS absensi;
DROP TABLE IF EXISTS jadwal_shift;
DROP TABLE IF EXISTS shift;
DROP TABLE IF EXISTS riwayat_upah;
DROP TABLE IF EXISTS pekerja;
DROP TABLE IF EXISTS sesi;
DROP TABLE IF EXISTS pengguna;
DROP TABLE IF EXISTS produk;

SET FOREIGN_KEY_CHECKS = 1;

-- =============================================================================
-- 0. TABEL BAWAAN TEMPLATE AWAL (PRODUK)
-- =============================================================================
CREATE TABLE produk (
    id         INT AUTO_INCREMENT PRIMARY KEY,
    nama       VARCHAR(100) NOT NULL,
    harga      INT NOT NULL DEFAULT 0,
    stok       INT NOT NULL DEFAULT 0,
    keterangan TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL,
    deleted_at TIMESTAMP NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- =============================================================================
-- 1. MODUL AUTENTIKASI & HAK AKSES (PENGGUNA & SESI) - Anggota 1 (Naufal)
-- =============================================================================
CREATE TABLE pengguna (
    id_pengguna   INT AUTO_INCREMENT PRIMARY KEY,
    username      VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    nama_lengkap  VARCHAR(100) NOT NULL,
    -- 3 Peran konsisten:
    -- 'admin'      : Admin Penggajian
    -- 'supervisor' : Petugas Absensi / Supervisor Shift
    -- 'pemilik'    : Pemilik Konveksi (Owner)
    peran         ENUM('admin', 'supervisor', 'pemilik') NOT NULL DEFAULT 'admin',
    aktif         TINYINT(1) NOT NULL DEFAULT 1,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP NULL ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE sesi (
    token       VARCHAR(64) PRIMARY KEY,
    id_pengguna INT NOT NULL,
    expired_at  DATETIME NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_sesi_pengguna
        FOREIGN KEY (id_pengguna) REFERENCES pengguna(id_pengguna)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- =============================================================================
-- 2. MODUL MASTER PEKERJA & RIWAYAT UPAH (AB 02) - Anggota 2 (Cahya)
-- =============================================================================
CREATE TABLE pekerja (
    id_pekerja     INT AUTO_INCREMENT PRIMARY KEY,
    nama           VARCHAR(100) NOT NULL,
    jenis_pekerja  ENUM('harian', 'bulanan') NOT NULL,
    jabatan        VARCHAR(100) NOT NULL,
    kontak         VARCHAR(50) DEFAULT NULL,
    tanggal_masuk  DATE NOT NULL,
    tanggal_keluar DATE DEFAULT NULL,
    created_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP NULL ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT chk_pekerja_tanggal
        CHECK (tanggal_keluar IS NULL OR tanggal_keluar >= tanggal_masuk)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE riwayat_upah (
    id_upah       INT AUTO_INCREMENT PRIMARY KEY,
    id_pekerja    INT NOT NULL,
    berlaku_mulai DATE NOT NULL,
    nominal       DECIMAL(15, 2) NOT NULL,
    keterangan    VARCHAR(255) DEFAULT NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_riwayat_upah_pekerja
        FOREIGN KEY (id_pekerja) REFERENCES pekerja(id_pekerja)
        ON DELETE RESTRICT,
    CONSTRAINT uk_pekerja_berlaku
        UNIQUE (id_pekerja, berlaku_mulai),
    CONSTRAINT chk_nominal_upah_positif
        CHECK (nominal > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- =============================================================================
-- 3. MODUL JADWAL SHIFT & PENCATATAN ABSENSI (AB 03, AB 06, AB 07) - Anggota 3 (Falih)
-- =============================================================================
CREATE TABLE shift (
    id_shift    INT AUTO_INCREMENT PRIMARY KEY,
    nama_shift  VARCHAR(50) NOT NULL UNIQUE,
    jam_mulai   TIME NOT NULL,
    jam_selesai TIME NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE jadwal_shift (
    id_jadwal  INT AUTO_INCREMENT PRIMARY KEY,
    id_pekerja INT NOT NULL,
    id_shift   INT NOT NULL,
    tanggal    DATE NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_jadwal_pekerja
        FOREIGN KEY (id_pekerja) REFERENCES pekerja(id_pekerja)
        ON DELETE RESTRICT,
    CONSTRAINT fk_jadwal_shift
        FOREIGN KEY (id_shift) REFERENCES shift(id_shift)
        ON DELETE RESTRICT,
    -- AB 03: Maksimal satu jadwal shift untuk setiap pekerja per tanggal
    CONSTRAINT uk_jadwal_pekerja_tanggal
        UNIQUE (id_pekerja, tanggal)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE absensi (
    id_absensi       INT AUTO_INCREMENT PRIMARY KEY,
    id_jadwal        INT NOT NULL UNIQUE, -- 1 jadwal maksimal memiliki 1 catatan absensi
    status_kehadiran ENUM('hadir', 'tidak_hadir', 'perlu_diperiksa') NOT NULL DEFAULT 'perlu_diperiksa',
    jam_masuk        TIME DEFAULT NULL,
    jam_keluar       TIME DEFAULT NULL,
    menit_terlambat  INT NOT NULL DEFAULT 0,
    menit_lembur     INT NOT NULL DEFAULT 0,
    catatan          VARCHAR(255) DEFAULT NULL,
    created_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP NULL ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_absensi_jadwal
        FOREIGN KEY (id_jadwal) REFERENCES jadwal_shift(id_jadwal)
        ON DELETE RESTRICT,
    -- AB 03: Jam keluar tidak boleh mendahului jam masuk (tidak ada shift lintas tengah malam)
    CONSTRAINT chk_absensi_jam
        CHECK (jam_masuk IS NULL OR jam_keluar IS NULL OR jam_keluar >= jam_masuk),
    CONSTRAINT chk_absensi_non_negatif
        CHECK (menit_terlambat >= 0 AND menit_lembur >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- =============================================================================
-- 4. MODUL PERIODE & ENGINE PERHITUNGAN GAJI (AB 01, AB 04, AB 05, AB 09, AB 10) - Anggota 4 (Daffa)
-- =============================================================================
CREATE TABLE periode_gaji (
    id_periode         INT AUTO_INCREMENT PRIMARY KEY,
    jenis_periode      ENUM('mingguan', 'bulanan') NOT NULL,
    tanggal_mulai      DATE NOT NULL,
    tanggal_akhir      DATE NOT NULL,
    tanggal_pembayaran DATE NOT NULL,
    status             ENUM('terbuka', 'tertutup') NOT NULL DEFAULT 'terbuka',
    ditutup_oleh       INT DEFAULT NULL,
    waktu_tutup        DATETIME DEFAULT NULL,
    created_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         TIMESTAMP NULL ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_periode_penutup
        FOREIGN KEY (ditutup_oleh) REFERENCES pengguna(id_pengguna)
        ON DELETE SET NULL,
    CONSTRAINT chk_periode_tanggal
        CHECK (tanggal_akhir >= tanggal_mulai)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE gaji (
    id_gaji            INT AUTO_INCREMENT PRIMARY KEY,
    id_periode         INT NOT NULL,
    id_pekerja         INT NOT NULL,
    gaji_pokok         DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    upah_lembur        DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    potongan_terlambat DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    potongan_kasbon    DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    koreksi_bersih     DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    gaji_bruto         DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    gaji_bersih        DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    status_bayar       ENUM('belum_dibayar', 'dibayar') NOT NULL DEFAULT 'belum_dibayar',
    waktu_bayar        DATETIME DEFAULT NULL,
    created_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         TIMESTAMP NULL ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_gaji_periode
        FOREIGN KEY (id_periode) REFERENCES periode_gaji(id_periode)
        ON DELETE RESTRICT,
    CONSTRAINT fk_gaji_pekerja
        FOREIGN KEY (id_pekerja) REFERENCES pekerja(id_pekerja)
        ON DELETE RESTRICT,
    -- ERD: Satu pekerja memiliki maksimal satu gaji per periode
    CONSTRAINT uk_gaji_periode_pekerja
        UNIQUE (id_periode, id_pekerja)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE detail_gaji (
    id_detail           INT AUTO_INCREMENT PRIMARY KEY,
    id_gaji             INT NOT NULL,
    id_upah             INT NOT NULL,
    id_absensi          INT NOT NULL UNIQUE, -- Menyimpan rincian hari kerja terjadwal (tidak dihitung ganda)
    tanggal             DATE NOT NULL,
    upah_harian_berlaku DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    upah_per_jam        DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    menit_terlambat     INT NOT NULL DEFAULT 0,
    potongan_hari_ini   DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    jam_lembur          DECIMAL(6, 2) NOT NULL DEFAULT 0.00,
    lembur_hari_ini     DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    created_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_detail_gaji
        FOREIGN KEY (id_gaji) REFERENCES gaji(id_gaji)
        ON DELETE CASCADE,
    CONSTRAINT fk_detail_upah
        FOREIGN KEY (id_upah) REFERENCES riwayat_upah(id_upah)
        ON DELETE RESTRICT,
    CONSTRAINT fk_detail_absensi
        FOREIGN KEY (id_absensi) REFERENCES absensi(id_absensi)
        ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- =============================================================================
-- 5. MODUL KASBON & KOREKSI ANTARPERIODE (AB 08, AB 11) - Anggota 5 (Adel)
-- =============================================================================
CREATE TABLE kasbon (
    id_kasbon       INT AUTO_INCREMENT PRIMARY KEY,
    id_pekerja      INT NOT NULL,
    tanggal_pinjam  DATE NOT NULL,
    jumlah          DECIMAL(15, 2) NOT NULL,
    total_terpotong DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    saldo_sisa      DECIMAL(15, 2) NOT NULL,
    status          ENUM('aktif', 'lunas') NOT NULL DEFAULT 'aktif',
    keterangan      VARCHAR(255) DEFAULT NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NULL ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_kasbon_pekerja
        FOREIGN KEY (id_pekerja) REFERENCES pekerja(id_pekerja)
        ON DELETE RESTRICT,
    CONSTRAINT chk_kasbon_jumlah
        CHECK (jumlah > 0 AND total_terpotong >= 0 AND saldo_sisa >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE potongan_kasbon (
    id_potongan INT AUTO_INCREMENT PRIMARY KEY,
    id_gaji     INT NOT NULL,
    id_kasbon   INT NOT NULL,
    jumlah      DECIMAL(15, 2) NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_potongan_gaji
        FOREIGN KEY (id_gaji) REFERENCES gaji(id_gaji)
        ON DELETE CASCADE,
    CONSTRAINT fk_potongan_kasbon
        FOREIGN KEY (id_kasbon) REFERENCES kasbon(id_kasbon)
        ON DELETE RESTRICT,
    CONSTRAINT uk_potongan_gaji_kasbon
        UNIQUE (id_gaji, id_kasbon),
    CONSTRAINT chk_jumlah_potongan_positif
        CHECK (jumlah > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE koreksi_gaji (
    id_koreksi     INT AUTO_INCREMENT PRIMARY KEY,
    id_gaji_asal   INT NOT NULL,
    -- id_gaji_tujuan bernilai NULL saat koreksi diajukan/disetujui,
    -- dan otomatis diisi saat perhitungan gaji periode berikutnya dijalankan (AB 11)
    id_gaji_tujuan INT DEFAULT NULL,
    dicatat_oleh   INT NOT NULL,
    disetujui_oleh INT DEFAULT NULL,
    alasan         VARCHAR(255) NOT NULL,
    nilai_selisih  DECIMAL(15, 2) NOT NULL, -- Positif menambah gaji tujuan, negatif mengurangi
    status         ENUM('diajukan', 'disetujui', 'diterapkan') NOT NULL DEFAULT 'diajukan',
    created_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP NULL ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_koreksi_gaji_asal
        FOREIGN KEY (id_gaji_asal) REFERENCES gaji(id_gaji)
        ON DELETE RESTRICT,
    CONSTRAINT fk_koreksi_gaji_tujuan
        FOREIGN KEY (id_gaji_tujuan) REFERENCES gaji(id_gaji)
        ON DELETE SET NULL,
    CONSTRAINT fk_koreksi_pencatat
        FOREIGN KEY (dicatat_oleh) REFERENCES pengguna(id_pengguna)
        ON DELETE RESTRICT,
    CONSTRAINT fk_koreksi_penyetuju
        FOREIGN KEY (disetujui_oleh) REFERENCES pengguna(id_pengguna)
        ON DELETE SET NULL,
    CONSTRAINT chk_koreksi_gaji_berbeda
        CHECK (id_gaji_tujuan IS NULL OR id_gaji_asal <> id_gaji_tujuan),
    CONSTRAINT chk_koreksi_selisih_tidak_nol
        CHECK (nilai_selisih <> 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- =============================================================================
-- DATA AWAL (SEEDER MASTER SHIFT) - AB 03
-- =============================================================================
INSERT INTO shift (nama_shift, jam_mulai, jam_selesai) VALUES
    ('Pagi', '07:00:00', '15:00:00'),
    ('Siang', '15:00:00', '23:00:00');
