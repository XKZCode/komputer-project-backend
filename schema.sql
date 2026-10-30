CREATE DATABASE IF NOT EXISTS ht_monitoring_db;
USE ht_monitoring_db;

-- Tabel Utama Transkrip Audio & Indikasi Bahaya
CREATE TABLE IF NOT EXISTS audio_transcripts (
    id INT AUTO_INCREMENT PRIMARY KEY,
    audio_path VARCHAR(255) NOT NULL,
    transcript_text TEXT NOT NULL,
    threat_level VARCHAR(50) DEFAULT 'Aman',
    keywords_detected VARCHAR(255) DEFAULT '-',
    latitude DECIMAL(10, 8) DEFAULT 0.0,
    longitude DECIMAL(11, 8) DEFAULT 0.0,
    review_status VARCHAR(50) DEFAULT 'Unreviewed',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabel Pengguna (Operator / Admin)
CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(100) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    role VARCHAR(50) DEFAULT 'operator',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Data Default User Login Demo BRS
INSERT IGNORE INTO users (username, password, role) VALUES 
('operator1', 'operator123', 'operator'),
('admin1', 'admin123', 'admin');