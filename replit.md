# Prediksi Toto Macau 5D

Sistem prediksi otomatis Toto Macau 5D dengan 3 metode prediksi (Paito, Shio, AI) yang menghasilkan 10 nomor 5D per sesi.

## Run & Operate

- `cd artifacts/toto-macau && go run .` — jalankan server Go (port dari PORT env var)
- Required env: `PORT` — port yang digunakan server (default 8080)
- Database: Replit PostgreSQL — koneksi via `DATABASE_URL` env var (auto-provisioned)

## Stack

- **Backend**: Go 1.25 + standard library (net/http)
- **Database**: SQLite via `modernc.org/sqlite` (pure Go, no CGO)
- **Frontend**: Embedded HTML template (single-page, no build step)
- **Routing**: Proxy routes `/` ke Go server di port 23007

## Where things live

- `artifacts/toto-macau/main.go` — server, route handlers, HTML serving
- `artifacts/toto-macau/db.go` — SQLite init, CRUD, session logic
- `artifacts/toto-macau/predict.go` — semua algoritma prediksi
- `artifacts/toto-macau/templates/index.html` — UI single-page (di-embed ke binary)
- `artifacts/toto-macau/toto.db` — database SQLite (auto-created)

## Architecture decisions

- Go server meng-embed HTML via `//go:embed templates/index.html` sehingga hanya 1 binary
- Routes menggunakan `/status`, `/predictions`, `/results`, `/history`, `/generate`, `/paito` (bukan `/api/`) karena proxy sudah route `/api` ke api-server Node.js
- SQLite `modernc.org/sqlite` dipilih karena pure Go, tidak perlu CGO
- Auto-prediksi dipanggil setelah setiap `POST /results` untuk sesi berikutnya

## Product

- Input hasil keluaran Toto Macau 5D (2 sesi per hari)
- Auto-generate prediksi untuk sesi berikutnya setelah hasil diinput
- 4 metode: Paito (analisis warna), Shio (zodiak), AI (statistik), Gabungan (10 nomor)
- Setiap nomor menampilkan Shio dan kode warna Merah/Hitam
- Tombol copy per nomor dan copy semua
- Tabel paito historis

## User preferences

_Populate as you build — explicit user instructions worth remembering across sessions._

## Gotchas

- Jangan gunakan path `/api/*` untuk Go routes — akan di-intercept api-server Node.js
- Setelah edit template HTML, perlu restart workflow agar perubahan aktif (embed bersifat compile-time)
- Database toto.db disimpan di direktori kerja saat server dijalankan

## Pointers

- See the `pnpm-workspace` skill for workspace structure
