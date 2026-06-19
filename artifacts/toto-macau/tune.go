package main

import (
        "fmt"
        "sort"
)

// PaitoConfig — parameter tunable untuk algoritma Paito Multi-Lag+Delta
type PaitoConfig struct {
        Lag2Weight   float64 `json:"lag2_weight"`
        Lag3Weight   float64 `json:"lag3_weight"`
        DeltaPoolMax int     `json:"delta_pool_max"`
        MLagPoolMax  int     `json:"mlag_pool_max"`
}

// activePaitoConfig — config aktif yang digunakan predictPaito()
// Default: nilai awal dari desain pertama. Diupdate setelah tuning.
var activePaitoConfig = PaitoConfig{
        Lag2Weight:   0.6,
        Lag3Weight:   0.3,
        DeltaPoolMax: 15,
        MLagPoolMax:  40,
}

type TuneConfigResult struct {
        Config PaitoConfig `json:"config"`
        Rate2D float64     `json:"rate_2d"`
        Rate3D float64     `json:"rate_3d"`
        Hit2D  int         `json:"hit_2d"`
        Total  int         `json:"total"`
}

type TuneResult struct {
        BestConfig  PaitoConfig        `json:"best_config"`
        PrevConfig  PaitoConfig        `json:"prev_config"`
        BestRate2D  float64            `json:"best_rate_2d"`
        PrevRate2D  float64            `json:"prev_rate_2d"`
        Improved    float64            `json:"improved"`
        Combos      int                `json:"combos"`
        Tested      int                `json:"tested"`
        TopResults  []TuneConfigResult `json:"top_results"`
        Note        string             `json:"note"`
}

// tunePaitoConfig menjalankan grid search atas parameter Paito,
// memilih konfigurasi terbaik berdasarkan hit rate 2D pada data historis,
// lalu mengupdate activePaitoConfig.
func tunePaitoConfig() TuneResult {
        all := getRecentResults(200)
        if len(all) < 15 {
                return TuneResult{Note: "Butuh minimal 15 hasil untuk tuning. Masukkan lebih banyak data dulu."}
        }

        // Reverse ke oldest-first untuk simulasi kronologis
        for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
                all[i], all[j] = all[j], all[i]
        }

        // Grid parameter yang dicoba
        lag2Opts  := []float64{0.3, 0.5, 0.7, 0.9}
        lag3Opts  := []float64{0.1, 0.2, 0.4}
        deltaOpts := []int{5, 10, 15}
        mlagOpts  := []int{30, 40}

        minHistory := 8
        var allResults []TuneConfigResult

        for _, w2 := range lag2Opts {
                for _, w3 := range lag3Opts {
                        for _, dp := range deltaOpts {
                                for _, mp := range mlagOpts {
                                        cfg := PaitoConfig{
                                                Lag2Weight:   w2,
                                                Lag3Weight:   w3,
                                                DeltaPoolMax: dp,
                                                MLagPoolMax:  mp,
                                        }
                                        hit2d, hit3d, total := 0, 0, 0
                                        for i := minHistory; i < len(all); i++ {
                                                actual := all[i].Nomor
                                                for len(actual) < 4 {
                                                        actual = "0" + actual
                                                }
                                                history := make([]Result, i)
                                                for k := 0; k < i; k++ {
                                                        history[k] = all[i-1-k]
                                                }
                                                preds := predictPaitoWithConfig(history, cfg)
                                                total++
                                                h2d, h3d := false, false
                                                for _, p := range preds {
                                                        for len(p) < 4 {
                                                                p = "0" + p
                                                        }
                                                        if p[2:] == actual[2:] {
                                                                h2d = true
                                                        }
                                                        if p[1:] == actual[1:] {
                                                                h3d = true
                                                        }
                                                }
                                                if h2d {
                                                        hit2d++
                                                }
                                                if h3d {
                                                        hit3d++
                                                }
                                        }
                                        r2, r3 := 0.0, 0.0
                                        if total > 0 {
                                                r2 = float64(hit2d) / float64(total) * 100
                                                r3 = float64(hit3d) / float64(total) * 100
                                        }
                                        allResults = append(allResults, TuneConfigResult{
                                                Config: cfg,
                                                Rate2D: r2,
                                                Rate3D: r3,
                                                Hit2D:  hit2d,
                                                Total:  total,
                                        })
                                }
                        }
                }
        }

        // Sort by rate_2d descending
        sort.Slice(allResults, func(i, j int) bool {
                return allResults[i].Rate2D > allResults[j].Rate2D
        })

        // Hitung rate config sebelumnya
        prevCfg := activePaitoConfig
        prevRate2D := 0.0
        for _, r := range allResults {
                if r.Config == prevCfg {
                        prevRate2D = r.Rate2D
                        break
                }
        }
        // Jika config sebelumnya tidak ada di grid (nilai custom), hitung langsung
        if prevRate2D == 0 && len(all) > minHistory {
                hit2d, total := 0, 0
                for i := minHistory; i < len(all); i++ {
                        actual := all[i].Nomor
                        for len(actual) < 4 {
                                actual = "0" + actual
                        }
                        history := make([]Result, i)
                        for k := 0; k < i; k++ {
                                history[k] = all[i-1-k]
                        }
                        preds := predictPaitoWithConfig(history, prevCfg)
                        total++
                        for _, p := range preds {
                                for len(p) < 4 {
                                        p = "0" + p
                                }
                                if p[2:] == actual[2:] {
                                        hit2d++
                                        break
                                }
                        }
                }
                if total > 0 {
                        prevRate2D = float64(hit2d) / float64(total) * 100
                }
        }

        best := allResults[0]
        activePaitoConfig = best.Config
        // Simpan ke riwayat tuning
        go func(r TuneResult) { saveTuneHistory(r) }(TuneResult{
                BestConfig: best.Config,
                PrevConfig: prevCfg,
                BestRate2D: best.Rate2D,
                PrevRate2D: prevRate2D,
                Improved:   best.Rate2D - prevRate2D,
                Combos:     len(allResults),
                Tested:     best.Total,
        })

        top := allResults
        if len(top) > 10 {
                top = top[:10]
        }

        tested := 0
        if len(allResults) > 0 {
                tested = allResults[0].Total
        }

        return TuneResult{
                BestConfig: best.Config,
                PrevConfig: prevCfg,
                BestRate2D: best.Rate2D,
                PrevRate2D: prevRate2D,
                Improved:   best.Rate2D - prevRate2D,
                Combos:     len(allResults),
                Tested:     tested,
                TopResults: top,
                Note: fmt.Sprintf(
                        "Grid search %d kombinasi parameter × %d sesi historis — config terbaik diterapkan otomatis",
                        len(allResults), tested,
                ),
        }
}
