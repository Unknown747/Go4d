package main

import (
	"fmt"
	"sort"
)

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

	methodKeys := []string{"PAITO", "SHIO", "AI", "EKORAS", "KOPKEP", "MATH", "GABUNGAN"}
	methodLabels := map[string]string{
		"PAITO":    "Paito",
		"SHIO":     "Shio",
		"AI":       "AI",
		"EKORAS":   "AS/Ekor",
		"KOPKEP":   "Kop·Kep",
		"MATH":     "Rumus Math",
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
			"EKORAS":   predictEkorAS(history),
			"KOPKEP":   predictKopKepala(history),
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
