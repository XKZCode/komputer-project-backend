Emergency Audio Monitoring System (Backend Microservices)

Sistem backend berbasis mikroservis untuk pemantauan insiden darurat melalui sinyal/audio Handie Talkie (HT). Sistem ini menerima unggahan berkas audio, melakukan transkripsi suara secara otomatis menggunakan modul Python (Whisper STT), menganalisis tingkat bahaya insiden, dan menyimpannya ke database MySQL.

Arsitektur Sistem

Proyek ini terdiri dari 3 layanan utama yang di-container-isasi menggunakan Docker:

1. Go Backend (REST API) (`Port 8080`)
   - Dibangun menggunakan **Go (Gin Framework)**.
   - Mengelola endpoint API insiden, pemrosesan bisnis, upload berkas audio, dan ekspor data CSV.
   - Menyajikan panel pengetesan berbasis web statis (`/tester`).

2. Python STT Service (`Port 8000`)
   - Dibangun menggunakan **Python (FastAPI & Whisper)**.
   - Menerima sampel audio dan mengubah suara (*speech-to-text*) menjadi teks secara otomatis.

3. MySQL Database (`Port 3306`)
   - Menyimpan seluruh catatan log insiden, tingkat bahaya, kata kunci, lokasi GPS, dan status penanganan.

---

Cara Menjalankan Proyek (Prasyarat: Docker Desktop)

1. Clone Repositori
```bash
git clone https://github.com/XKZCode/komputer-project-backend.git
cd komputer-project-backend
