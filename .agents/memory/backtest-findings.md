---
name: Backtest findings & method changes
description: Hasil walk-forward backtest 161 sesi dan perubahan metode prediksi yang diterapkan
---

## Hasil Backtest (161 sesi, mulai dari index ke-20)

| Metode | Hit 2D | Rate 2D | Rate Shio | avg# |
|--------|--------|---------|-----------|------|
| Gabungan | 23 | 14.3% | 78.3% | 20 |
| Shio | 7 | 4.3% | 13.0% | 5 |
| Paito | 5 | 3.1% | 37.9% | 5 |
| PolaEkor (lama) | 5 | 3.1% | 14.9% | 5 |
| Gap(AI) | 4 | 2.5% | 44.7% | 5 |
| Matrix | 4 | 2.5% | 41.0% | 5 |

Tidak ada metode yang hit 4D exact dalam 161 sesi.

## Pola data statistik (181 hasil, 20 Mei – 19 Jun 2026)

- **Top digit AS**: 1(27), 6(24), 2(20), 3(20)
- **Top ekor 2D**: 84(6x), 00(6x), 57(5x), 27(5x), 40(4x), 78(4x)
- **Digit paling sering (semua posisi)**: 2(12.3%), 4(10.9%), 0(10.8%), 7(10.6%)
- **Nomor 4D yang muncul 2x**: 3940, 8200, 7868 — bukti nomor BISA ulang
- **Warna ekor**: Hitam(genap) 53%, Merah(ganjil) 47%

## Perubahan yang diterapkan

**Dihapus:**
- `predictEkorAS` / EKORAS — performa terburuk: rate shio 14.9% (hampir random), 2D tidak lebih baik dari metode lain
- KOPKEP — tidak pernah berjalan dengan data yang cukup

**Ditambahkan:**
- `predictHotEkor` / HOTEKOR — fokus pada ekor 2D yang TERBUKTI sering muncul (frekuensi murni), window pendek 30 sesi diberi bobot 2.5x lebih besar. Berbeda dari EKORAS yang pakai overdue/gap.

**Diperbaiki:**
- `filterPastResults` — sebelumnya memfilter SEMUA histori (100 sesi), sekarang hanya filter 5 sesi terakhir. **Why:** nomor bisa muncul ulang (3940, 8200, 7868 terbukti), memfilter terlalu luas justru membuang kandidat yang valid.

## Referensi file

- `artifacts/toto-macau/predict.go` — `predictHotEkor`, `filterPastResults`, `predictGabungan`
- `artifacts/toto-macau/main.go` — `generateAndSavePredictions`, `handleRekomendasi`, `handleGetPredictions`
- `artifacts/toto-macau/backtest.go` — method keys dan labels
- `artifacts/toto-macau/templates/index.html` — tab "Hot Ekor" menggantikan "Pola Ekor"
