-- =============================================================================
-- SCHEMA DATABASE: Sistem Penggajian Pekerja Harian dan Shift
-- Studi Kasus: Konveksi Mitra Jaya (Kelompok 3)
-- Engine: MySQL / MariaDB (InnoDB, utf8mb4)
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
    id         BIGINT AUTO_INCREMENT PRIMARY KEY,
    nama       VARCHAR(100) NOT NULL,
    harga      INT NOT NULL DEFAULT 0,
    stok       INT NOT NULL DEFAULT 0,
    keterangan TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL,
    deleted_at TIMESTAMP NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- =============================================================================
-- 1. MODUL AUTENTIKASI & HAK AKSES (PENGGUNA & SESI)
-- =============================================================================
CREATE TABLE pengguna (
    id_pengguna   BIGINT AUTO_INCREMENT PRIMARY KEY,
    username      VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    nama_lengkap  VARCHAR(100) NOT NULL,
    peran         ENUM('admin', 'pemilik', 'petugas_absensi') NOT NULL DEFAULT 'admin',
    aktif         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP NULL ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE sesi (
    id_sesi        BIGINT AUTO_INCREMENT PRIMARY KEY,
    id_pengguna    BIGINT NOT NULL,
    token          VARCHAR(128) NOT NULL UNIQUE,
    kedaluwarsa_at TIMESTAMP NOT NULL,
    created_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_sesi_pengguna
        FOREIGN KEY (id_pengguna) REFERENCES pengguna(id_pengguna)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- =============================================================================
-- 2. MODUL MASTER PEKERJA & RIWAYAT UPAH (AB 02)
-- =============================================================================
CREATE TABLE pekerja (
    id_pekerja     BIGINT AUTO_INCREMENT PRIMARY KEY,
    nama_pekerja   VARCHAR(100) NOT NULL,
    jenis_pekerja  ENUM('harian', 'bulanan') NOT NULL,
    jabatan        VARCHAR(100) NOT NULL,
    kontak         VARCHAR(50) NULL,
    tanggal_masuk  DATE NOT NULL,
    tanggal_keluar DATE NULL,
    status_aktif   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP NULL ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT chk_pekerja_tanggal
        CHECK (tanggal_keluar IS NULL OR tanggal_keluar >= tanggal_masuk)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE riwayat_upah (
    id_upah       BIGINT AUTO_INCREMENT PRIMARY KEY,
    id_pekerja    BIGINT NOT NULL,
    nominal_upah  DECIMAL(15, 2) NOT NULL,
    berlaku_mulai DATE NOT NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_riwayat_upah_pekerja
        FOREIGN KEY (id_pekerja) REFERENCES pekerja(id_pekerja)
        ON DELETE RESTRICT,
    CONSTRAINT uq_pekerja_berlaku_mulai
        UNIQUE (id_pekerja, berlaku_mulai),
    CONSTRAINT chk_nominal_upah_positif
        CHECK (nominal_upah > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- =============================================================================
-- 3. MODUL JADWAL SHIFT & PENCATATAN ABSENSI (AB 03, AB 06, AB 07)
-- =============================================================================
CREATE TABLE shift (
    id_shift   BIGINT AUTO_INCREMENT PRIMARY KEY,
    nama_shift VARCHAR(50) NOT NULL UNIQUE,
    jam_mulai  TIME NOT NULL,
    jam_selesai TIME NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE jadwal_shift (
    id_jadwal     BIGINT AUTO_INCREMENT PRIMARY KEY,
    id_pekerja    BIGINT NOT NULL,
    id_shift      BIGINT NOT NULL,
    tanggal_kerja DATE NOT NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP NULL ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_jadwal_pekerja
        FOREIGN KEY (id_pekerja) REFERENCES pekerja(id_pekerja)
        ON DELETE RESTRICT,
    CONSTRAINT fk_jadwal_shift
        FOREIGN KEY (id_shift) REFERENCES shift(id_shift)
        ON DELETE RESTRICT,
    -- AB 03: Maksimal satu jadwal shift untuk setiap pekerja per tanggal
    CONSTRAINT uq_jadwal_pekerja_tanggal
        UNIQUE (id_pekerja, tanggal_kerja)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE absensi (
    id_absensi       BIGINT AUTO_INCREMENT PRIMARY KEY,
    id_jadwal        BIGINT NOT NULL UNIQUE, -- 1 jadwal maksimal memiliki 1 catatan absensi
    status_hadir     ENUM('hadir', 'tidak_hadir', 'perlu_diperiksa') NOT NULL DEFAULT 'perlu_diperiksa',
    jam_masuk        TIME NULL,
    jam_keluar       TIME NULL,
    menit_terlambat  INT NOT NULL DEFAULT 0,
    jam_lembur       DECIMAL(6, 2) NOT NULL DEFAULT 0.00,
    perlu_diperiksa  BOOLEAN NOT NULL DEFAULT FALSE,
    catatan          TEXT NULL,
    created_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP NULL ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_absensi_jadwal
        FOREIGN KEY (id_jadwal) REFERENCES jadwal_shift(id_jadwal)
        ON DELETE RESTRICT,
    -- AB 03: Jam keluar tidak boleh mendahului jam masuk
    CONSTRAINT chk_absensi_jam
        CHECK (jam_masuk IS NULL OR jam_keluar IS NULL OR jam_keluar >= jam_masuk),
    CONSTRAINT chk_absensi_non_negatif
        CHECK (menit_terlambat >= 0 AND jam_lembur >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- =============================================================================
-- 4. MODUL PERIODE & ENGINE PERHITUNGAN GAJI (AB 01, AB 04, AB 05, AB 09, AB 10)
-- =============================================================================
CREATE TABLE periode_gaji (
    id_periode      BIGINT AUTO_INCREMENT PRIMARY KEY,
    jenis_periode   ENUM('mingguan', 'bulanan') NOT NULL,
    tanggal_mulai   DATE NOT NULL,
    tanggal_selesai DATE NOT NULL,
    tanggal_bayar   DATE NULL,
    status_periode  ENUM('terbuka', 'ditutup') NOT NULL DEFAULT 'terbuka',
    ditutup_oleh    BIGINT NULL,
    ditutup_at      TIMESTAMP NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NULL ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_periode_penutup
        FOREIGN KEY (ditutup_oleh) REFERENCES pengguna(id_pengguna)
        ON DELETE SET NULL,
    CONSTRAINT chk_periode_tanggal
        CHECK (tanggal_selesai >= tanggal_mulai)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE gaji (
    id_gaji                 BIGINT AUTO_INCREMENT PRIMARY KEY,
    id_periode              BIGINT NOT NULL,
    id_pekerja              BIGINT NOT NULL,
    total_hadir             INT NOT NULL DEFAULT 0,
    total_gaji_pokok        DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    total_lembur            DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    gaji_bruto              DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    total_potongan_terlambat DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    total_potongan_kasbon   DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    total_koreksi           DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    gaji_bersih             DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    status_bayar            ENUM('draf', 'ditahan', 'dibayar') NOT NULL DEFAULT 'draf',
    waktu_bayar             TIMESTAMP NULL,
    created_at              TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMP NULL ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_gaji_periode
        FOREIGN KEY (id_periode) REFERENCES periode_gaji(id_periode)
        ON DELETE RESTRICT,
    CONSTRAINT fk_gaji_pekerja
        FOREIGN KEY (id_pekerja) REFERENCES pekerja(id_pekerja)
        ON DELETE RESTRICT,
    -- ERD: Satu pekerja memiliki maksimal satu gaji per periode
    CONSTRAINT uq_gaji_periode_pekerja
        UNIQUE (id_periode, id_pekerja)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE detail_gaji (
    id_detail          BIGINT AUTO_INCREMENT PRIMARY KEY,
    id_gaji            BIGINT NOT NULL,
    id_absensi         BIGINT NULL UNIQUE, -- Satu absensi digunakan maksimal pada satu rincian
    id_upah            BIGINT NOT NULL,
    tanggal_rincian    DATE NOT NULL,
    upah_pokok_hari    DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    upah_per_jam       DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    upah_lembur        DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    potongan_terlambat DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    created_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_detail_gaji
        FOREIGN KEY (id_gaji) REFERENCES gaji(id_gaji)
        ON DELETE CASCADE,
    CONSTRAINT fk_detail_absensi
        FOREIGN KEY (id_absensi) REFERENCES absensi(id_absensi)
        ON DELETE RESTRICT,
    CONSTRAINT fk_detail_upah
        FOREIGN KEY (id_upah) REFERENCES riwayat_upah(id_upah)
        ON DELETE RESTRICT,
    CONSTRAINT uq_detail_gaji_tanggal
        UNIQUE (id_gaji, tanggal_rincian)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- =============================================================================
-- 5. MODUL KASBON & KOREKSI ANTARPERIODE (AB 08, AB 11)
-- =============================================================================
CREATE TABLE kasbon (
    id_kasbon       BIGINT AUTO_INCREMENT PRIMARY KEY,
    id_pekerja      BIGINT NOT NULL,
    tanggal_pinjam  DATE NOT NULL,
    jumlah_pinjaman DECIMAL(15, 2) NOT NULL,
    total_potongan  DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    saldo_kasbon    DECIMAL(15, 2) NOT NULL,
    status_kasbon   ENUM('belum_lunas', 'lunas') NOT NULL DEFAULT 'belum_lunas',
    keterangan      TEXT NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NULL ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_kasbon_pekerja
        FOREIGN KEY (id_pekerja) REFERENCES pekerja(id_pekerja)
        ON DELETE RESTRICT,
    CONSTRAINT chk_kasbon_jumlah
        CHECK (jumlah_pinjaman > 0 AND total_potongan >= 0 AND saldo_kasbon >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE potongan_kasbon (
    id_potongan     BIGINT AUTO_INCREMENT PRIMARY KEY,
    id_kasbon       BIGINT NOT NULL,
    id_gaji         BIGINT NOT NULL,
    jumlah_potongan DECIMAL(15, 2) NOT NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_potongan_kasbon
        FOREIGN KEY (id_kasbon) REFERENCES kasbon(id_kasbon)
        ON DELETE RESTRICT,
    CONSTRAINT fk_potongan_gaji
        FOREIGN KEY (id_gaji) REFERENCES gaji(id_gaji)
        ON DELETE CASCADE,
    CONSTRAINT uq_potongan_kasbon_gaji
        UNIQUE (id_kasbon, id_gaji),
    CONSTRAINT chk_jumlah_potongan_positif
        CHECK (jumlah_potongan > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE koreksi_gaji (
    id_koreksi        BIGINT AUTO_INCREMENT PRIMARY KEY,
    id_gaji_asal      BIGINT NOT NULL,
    id_gaji_tujuan    BIGINT NOT NULL,
    dicatat_oleh      BIGINT NOT NULL,
    disetujui_oleh    BIGINT NULL,
    alasan            TEXT NOT NULL,
    nilai_selisih     DECIMAL(15, 2) NOT NULL, -- Positif menambah gaji tujuan, negatif mengurangi
    status_persetujuan ENUM('menunggu', 'disetujui', 'ditolak') NOT NULL DEFAULT 'menunggu',
    disetujui_at      TIMESTAMP NULL,
    created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP NULL ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_koreksi_gaji_asal
        FOREIGN KEY (id_gaji_asal) REFERENCES gaji(id_gaji)
        ON DELETE RESTRICT,
    CONSTRAINT fk_koreksi_gaji_tujuan
        FOREIGN KEY (id_gaji_tujuan) REFERENCES gaji(id_gaji)
        ON DELETE RESTRICT,
    CONSTRAINT fk_koreksi_pencatat
        FOREIGN KEY (dicatat_oleh) REFERENCES pengguna(id_pengguna)
        ON DELETE RESTRICT,
    CONSTRAINT fk_koreksi_penyetuju
        FOREIGN KEY (disetujui_oleh) REFERENCES pengguna(id_pengguna)
        ON DELETE SET NULL,
    CONSTRAINT chk_koreksi_gaji_berbeda
        CHECK (id_gaji_asal <> id_gaji_tujuan),
    CONSTRAINT chk_koreksi_selisih_tidak_nol
        CHECK (nilai_selisih <> 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- =============================================================================
-- DATA AWAL (SEEDER MASTER SHIFT) - AB 03
-- =============================================================================
INSERT INTO shift (nama_shift, jam_mulai, jam_selesai) VALUES
    ('Pagi', '07:00:00', '15:00:00'),
    ('Siang', '15:00:00', '23:00:00');
