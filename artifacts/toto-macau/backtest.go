package main

import (
        "fmt"
        "sort"
)

// ─── BB CAMPURAN BACKTEST ────────────────────────────────────

type BBBacktestReport struct {
        Total  int     `json:"total"`
        Hit4D  int     `json:"hit_4d"`
        Hit3D  int     `json:"hit_3d"`
        Hit2D  int     `json:"hit_2d"`
        Rate4D float64 `json:"rate_4d"`
        Rate3D float64 `json:"rate_3d"`
        Rate2D float64 `json:"rate_2d"`
        // Baseline: P(6,k)/10000 * 100 for comparison
        Base4D float64 `json:"base_4d"`
        Base3D float64 `json:"base_3d"`
        Base2D float64 `json:"base_2d"`
        Note   string  `json:"note"`
}

// runBBBacktest: for each historical result (with min 10 prior draws),
// generate the 6-digit BB set from prior history, then check whether
// the actual result's digits are fully covered by that set.
// Coverage = all N digits of actual appear in the 6 BB digits.
func runBBBacktest() BBBacktestReport {
        all := getRecentResults(200)
        if len(all) < 15 {
                return BBBacktestReport{Note: "Butuh minimal 15 hasil untuk back-test BB. Masukkan lebih banyak data dulu."}
        }

        // Reverse to oldest-first
        for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
                all[i], all[j] = all[j], all[i]
        }

        minHistory := 10
        total, hit4d, hit3d, hit2d := 0, 0, 0, 0

        for i := minHistory; i < len(all); i++ {
                target := all[i]
                actual := target.Nomor
                for len(actual) < 4 {
                        actual = "0" + actual
                }

                // Build history newest-first
                history := make([]Result, i)
                for k := 0; k < i; k++ {
                        history[k] = all[i-1-k]
                }

                bbResult := predictBBFS(history, 6)
                bbSet := map[int]bool{}
                for _, d := range bbResult.BBDigits {
                        bbSet[d] = true
                }

                actualDigits := parse4D(actual)

                // 4D: all 4 digits in BB set
                all4In := true
                for _, d := range actualDigits {
                        if !bbSet[d] {
                                all4In = false
                                break
                        }
                }

                // 3D: last 3 digits (pos 1,2,3) in BB set
                all3In := true
                for _, d := range actualDigits[1:] {
                        if !bbSet[d] {
                                all3In = false
                                break
                        }
                }

                // 2D: last 2 digits (pos 2,3) in BB set
                all2In := true
                for _, d := range actualDigits[2:] {
                        if !bbSet[d] {
                                all2In = false
                                break
                        }
                }

                total++
                if all4In {
                        hit4d++
                }
                if all3In {
                        hit3d++
                }
                if all2In {
                        hit2d++
                }
        }

        // Random baseline: probability that k random digits from 0-9 are all within a random 6-digit set
        // P = C(6,k)/C(10,k) = (6!/(k!(6-k)!)) / (10!/(k!(10-k)!)) 
        // 4D: P(6,4)/P(10,4) = 360/5040 ≈ 7.14%  (actually 6*5*4*3 / 10*9*8*7)
        // 3D: 6*5*4 / 10*9*8 = 120/720 ≈ 16.67%
        // 2D: 6*5 / 10*9 = 30/90 ≈ 33.33%
        base4D := 360.0 / 5040.0 * 100
        base3D := 120.0 / 720.0 * 100
        base2D := 30.0 / 90.0 * 100

        r := BBBacktestReport{
                Total:  total,
                Hit4D:  hit4d,
                Hit3D:  hit3d,
                Hit2D:  hit2d,
                Base4D: base4D,
                Base3D: base3D,
                Base2D: base2D,
                Note:   fmt.Sprintf("Diuji %d sesi historis — cek apakah 6 digit BB mencakup semua digit hasil aktual", total),
        }
        if total > 0 {
                r.Rate4D = float64(hit4d) / float64(total) * 100
                r.Rate3D = float64(hit3d) / float64(total) * 100
                r.Rate2D = float64(hit2d) / float64(total) * 100
        }
        return r
}

type BacktestMethodResult struct {
        Method string  `json:"method"`
        Label  string  `json:"label"`
        Total  int     `json:"total"`
        Hit2D  int     `json:"hit_2d"`
        Hit3D  int     `json:"hit_3d"`
        Hit4D  int     `json:"hit_4d"`
        Rate2D float64 `json:"rate_2d"`
        Rate3D float64 `json:"rate_3d"`
        Rate4D float64 `json:"rate_4d"`
}

type BacktestReport struct {
        Methods    []BacktestMethodResult `json:"methods"`
        Tested     int                    `json:"tested"`
        HistoryLen int                    `json:"history_len"`
        Note       string                 `json:"note"`
}

func runBacktest() BacktestReport {
        all := getRecentResults(200)
        if len(all) < 6 {
                return BacktestReport{Note: "Butuh minimal 6 hasil untuk back-test. Masukkan lebih banyak data dulu."}
        }

        // all[] adalah newest-first, balik ke oldest-first
        for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
                all[i], all[j] = all[j], all[i]
        }

        type methodStats struct {
                total, hit2d, hit3d, hit4d int
        }

        methodKeys := []string{"PAITO", "SHIO", "AI", "HOTEKOR", "MATH", "GABUNGAN"}
        methodLabels := map[string]string{
                "PAITO":    "Paito",
                "SHIO":     "Shio",
                "AI":       "Gap Analysis",
                "HOTEKOR":  "Hot Ekor 2D",
                "MATH":     "Matrix",
                "GABUNGAN": "Gabungan",
        }
        stats := map[string]*methodStats{}
        for _, m := range methodKeys {
                stats[m] = &methodStats{}
        }

        tested := 0
        minHistory := 5

        for i := minHistory; i < len(all); i++ {
                target := all[i]
                actual := target.Nomor
                for len(actual) < 4 {
                        actual = "0" + actual
                }

                // History: all[0..i-1] dibalik jadi newest-first
                history := make([]Result, i)
                for k := 0; k < i; k++ {
                        history[k] = all[i-1-k]
                }

                actual2D := actual[2:]
                actual3D := actual[1:]

                methodPreds := map[string][]string{
                        "PAITO":    predictPaito(history),
                        "SHIO":     predictShio(history),
                        "AI":       predictAI(history),
                        "HOTEKOR":  predictHotEkor(history),
                        "MATH":     predictMath(history),
                        "GABUNGAN": predictGabungan(history),
                }

                tested++
                for m, preds := range methodPreds {
                        s := stats[m]
                        s.total++
                        h2d, h3d, h4d := false, false, false
                        for _, p := range preds {
                                for len(p) < 4 {
                                        p = "0" + p
                                }
                                if p == actual {
                                        h4d = true
                                }
                                if p[2:] == actual2D {
                                        h2d = true
                                }
                                if p[1:] == actual3D {
                                        h3d = true
                                }
                        }
                        if h2d {
                                s.hit2d++
                        }
                        if h3d {
                                s.hit3d++
                        }
                        if h4d {
                                s.hit4d++
                        }
                }
        }

        var results []BacktestMethodResult
        for _, m := range methodKeys {
                s := stats[m]
                r := BacktestMethodResult{
                        Method: m,
                        Label:  methodLabels[m],
                        Total:  s.total,
                        Hit2D:  s.hit2d,
                        Hit3D:  s.hit3d,
                        Hit4D:  s.hit4d,
                }
                if s.total > 0 {
                        r.Rate2D = float64(s.hit2d) / float64(s.total) * 100
                        r.Rate3D = float64(s.hit3d) / float64(s.total) * 100
                        r.Rate4D = float64(s.hit4d) / float64(s.total) * 100
                }
                results = append(results, r)
        }

        sort.Slice(results, func(i, j int) bool {
                return results[i].Rate2D > results[j].Rate2D
        })

        note := fmt.Sprintf("Diuji pada %d sesi (dari %d total, min 5 history sebelum tiap prediksi)", tested, len(all))
        return BacktestReport{
                Methods:    results,
                Tested:     tested,
                HistoryLen: len(all),
                Note:       note,
        }
}
