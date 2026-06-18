package main

import (
        "fmt"
        "math"
        "math/rand"
        "sort"
        "strconv"
        "strings"
)

var shioNames = []string{
        "Tikus", "Kerbau", "Macan", "Kelinci", "Naga", "Ular",
        "Kuda", "Kambing", "Monyet", "Ayam", "Anjing", "Babi",
}

// filterPastResults menghapus nomor yang sudah pernah keluar dari histori
func filterPastResults(nums []string, history []Result) []string {
        pastSet := map[string]bool{}
        for _, r := range history {
                n := r.Nomor
                for len(n) < 4 {
                        n = "0" + n
                }
                pastSet[n] = true
        }
        var filtered []string
        for _, n := range nums {
                n4 := n
                for len(n4) < 4 {
                        n4 = "0" + n4
                }
                if !pastSet[n4] {
                        filtered = append(filtered, n)
                }
        }
        return filtered
}

func shioOf(nomor string) string {
        if len(nomor) < 2 {
                return shioNames[0]
        }
        last2 := nomor[len(nomor)-2:]
        n, err := strconv.Atoi(last2)
        if err != nil {
                return shioNames[0]
        }
        return shioNames[n%12]
}

func warnaDigit(d int) string {
        if d%2 == 1 {
                return "M"
        }
        return "H"
}

func pad4(n int) string {
        return fmt.Sprintf("%04d", n%10000)
}

func parse4D(s string) [4]int {
        s = strings.TrimSpace(s)
        for len(s) < 4 {
                s = "0" + s
        }
        s = s[len(s)-4:]
        var d [4]int
        for i := 0; i < 4; i++ {
                d[i] = int(s[i] - '0')
        }
        return d
}

func colorCode4D(nomor string) string {
        for len(nomor) < 4 {
                nomor = "0" + nomor
        }
        nomor = nomor[len(nomor)-4:]
        codes := []string{}
        for _, c := range nomor {
                d := int(c - '0')
                if d%2 == 1 {
                        codes = append(codes, "M")
                } else {
                        codes = append(codes, "H")
                }
        }
        return strings.Join(codes, "")
}

// ============================================================
// diversifyPredictions: reorder candidates to maximise unique 2D ekor
// coverage while keeping higher-scored items first.
// ============================================================
func diversifyPredictions(nums []string, limit int) []string {
        seenEkor := map[string]bool{}
        var primary, secondary []string
        for _, n := range nums {
                n4 := n
                for len(n4) < 4 {
                        n4 = "0" + n4
                }
                ekor := n4[2:]
                if !seenEkor[ekor] {
                        seenEkor[ekor] = true
                        primary = append(primary, n)
                } else {
                        secondary = append(secondary, n)
                }
        }
        result := append(primary, secondary...)
        if len(result) > limit {
                result = result[:limit]
        }
        return result
}

// ============================================================
// Method 1: PAITO — frekuensi jangka menengah (30 sesi) per posisi
// Tidak terlalu "hot-biased": gunakan window panjang, 6 kandidat
// per posisi agar lebih tersebar, lalu diversify ekor.
// ============================================================
func predictPaito(history []Result) []string {
        if len(history) == 0 {
                return generateRandom(4, 0)
        }

        n := len(history)
        if n > 60 {
                n = 60
        }
        recent := history[:n]

        candidateDigits := [4][]int{}
        for pos := 0; pos < 4; pos++ {
                // Frekuensi dengan decay sangat lemah → semua sesi hampir sama bobotnya
                freq := [10]float64{}
                for k, r := range recent {
                        d := parse4D(r.Nomor)
                        w := math.Exp(-float64(k) * 0.03) // decay sangat lemah
                        freq[d[pos]] += w
                }

                // Gap: berapa lama digit tidak muncul di posisi ini
                lastSeen := [10]int{}
                for i := range lastSeen {
                        lastSeen[i] = n + 1
                }
                for k, r := range recent {
                        d := parse4D(r.Nomor)
                        if lastSeen[d[pos]] == n+1 {
                                lastSeen[d[pos]] = k
                        }
                }

                // Skor kombinasi: frekuensi 60% + overdue 40%
                type ds struct {
                        d     int
                        score float64
                }
                var ranked []ds
                for d := 0; d < 10; d++ {
                        gapScore := float64(lastSeen[d]) / float64(n+1)
                        score := freq[d]*0.6 + gapScore*0.4*5.0
                        ranked = append(ranked, ds{d, score})
                }
                sort.Slice(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })
                // 6 kandidat per posisi → lebih tersebar
                for i := 0; i < 6 && i < len(ranked); i++ {
                        candidateDigits[pos] = append(candidateDigits[pos], ranked[i].d)
                }
        }
        pool := combinePositions4D(candidateDigits, 50)
        return diversifyPredictions(pool, 5)
}

// ============================================================
// Method 2: SHIO — pilih 3 shio overdue, pasang dengan 3 kandidat
// digit AS dari frekuensi jangka menengah → 5 nomor beragam.
// ============================================================
func predictShio(history []Result) []string {
        if len(history) == 0 {
                return generateRandom(4, 1000)
        }

        n := len(history)
        if n > 60 {
                n = 60
        }

        // Hitung gap & frekuensi tiap shio
        shioGap := map[string]int{}
        shioFreq := map[string]int{}
        for i, r := range history[:n] {
                s := shioOf(r.Nomor)
                shioFreq[s]++
                if _, ok := shioGap[s]; !ok {
                        shioGap[s] = i // posisi pertama kali muncul = gap dari sekarang
                }
        }

        // Skor: gap besar (lama tidak muncul) dan frekuensi rendah = prioritas tinggi
        type ss struct {
                name  string
                score float64
        }
        var scores []ss
        for _, name := range shioNames {
                gap := n + 12 // default: belum pernah muncul
                if g, ok := shioGap[name]; ok {
                        gap = g
                }
                freq := shioFreq[name]
                score := float64(gap)*1.5 - float64(freq)*2.0
                scores = append(scores, ss{name, score})
        }
        sort.Slice(scores, func(i, j int) bool { return scores[i].score > scores[j].score })

        // Ambil 3 shio terbaik
        topShios := []string{}
        for i := 0; i < 3 && i < len(scores); i++ {
                topShios = append(topShios, scores[i].name)
        }

        // Digit AS: frekuensi jangka menengah (bukan hanya terbaru)
        asFreq := [10]float64{}
        for k, r := range history[:n] {
                d := parse4D(r.Nomor)
                w := math.Exp(-float64(k) * 0.04)
                asFreq[d[0]] += w
        }
        type df struct{ d int; f float64 }
        var asRanked []df
        for d := 0; d < 10; d++ {
                asRanked = append(asRanked, df{d, asFreq[d]})
        }
        sort.Slice(asRanked, func(i, j int) bool { return asRanked[i].f > asRanked[j].f })
        // Ambil 3 kandidat AS yang berbeda
        topAS := []int{}
        for i := 0; i < 3 && i < len(asRanked); i++ {
                topAS = append(topAS, asRanked[i].d)
        }

        seen := map[string]bool{}
        var results []string

        for _, shioName := range topShios {
                shioIdx := 0
                for i, s := range shioNames {
                        if s == shioName {
                                shioIdx = i
                                break
                        }
                }
                // Kumpulkan semua 2D ekor yang cocok shio ini (00-99)
                var ekorCands []int
                for v := 0; v <= 99; v++ {
                        if v%12 == shioIdx {
                                ekorCands = append(ekorCands, v)
                        }
                }
                // Pasang digit AS (top 3) dengan ekor kandidat (top 3)
                for _, asD := range topAS {
                        for _, e2 := range ekorCands[:min(len(ekorCands), 3)] {
                                // Nomor 4D: AS + mid1 + 2-digit ekor
                                // mid1 = digit tengah dari frekuensi, ambil dari asD untuk kesederhanaan
                                mid1 := (asD + e2/10) % 10
                                n4 := fmt.Sprintf("%d%d%02d", asD, mid1, e2)
                                for len(n4) < 4 {
                                        n4 = "0" + n4
                                }
                                if len(n4) > 4 {
                                        n4 = n4[len(n4)-4:]
                                }
                                if !seen[n4] && shioOf(n4) == shioName {
                                        seen[n4] = true
                                        results = append(results, n4)
                                }
                                if len(results) >= 5 {
                                        break
                                }
                        }
                        if len(results) >= 5 {
                                break
                        }
                }
                if len(results) >= 5 {
                        break
                }
        }

        for len(results) < 5 {
                rnd := generateRandom(1, 2000+len(results))
                if !seen[rnd[0]] {
                        seen[rnd[0]] = true
                        results = append(results, rnd[0])
                }
        }
        return results[:min(5, len(results))]
}

// ============================================================
// Method 3: GAP ANALYSIS — fokus digit yang overdue di tiap posisi
// berdasarkan rata-rata jarak kemunculan vs gap aktual saat ini.
// Lebih akurat dari Hot·Cold karena berbasis statistik gap, bukan
// sekadar frekuensi terbaru.
// ============================================================
func predictAI(history []Result) []string {
        if len(history) == 0 {
                return generateRandom(4, 3000)
        }

        n := len(history)
        if n > 60 {
                n = 60
        }
        recent := history[:n]

        // Untuk tiap posisi, hitung:
        // 1. Rata-rata jarak antar kemunculan tiap digit (avg gap)
        // 2. Gap aktual saat ini (berapa sesi sudah tidak muncul)
        // Score = (actual_gap - avg_gap) / avg_gap  →  makin positif = makin overdue

        candidateDigits := [4][]int{}
        for pos := 0; pos < 4; pos++ {
                // Kumpulkan posisi kemunculan tiap digit
                occurrences := [10][]int{}
                for k, r := range recent {
                        d := parse4D(r.Nomor)
                        occurrences[d[pos]] = append(occurrences[d[pos]], k)
                }

                // Hitung avg gap dan gap saat ini
                type ds struct {
                        d     int
                        score float64
                }
                var ranked []ds
                for d := 0; d < 10; d++ {
                        occ := occurrences[d]
                        if len(occ) == 0 {
                                // Tidak pernah muncul = sangat overdue
                                ranked = append(ranked, ds{d, float64(n) * 2.0})
                                continue
                        }
                        // Avg gap = jarak rata-rata antar kemunculan berurutan
                        avgGap := float64(n) / float64(len(occ))
                        // Gap aktual = berapa sesi sudah tidak muncul (posisi pertama = occ[0])
                        actualGap := float64(occ[0])
                        // Overduenya = seberapa lebih lama dari rata-rata
                        score := (actualGap - avgGap) / (avgGap + 1.0)
                        // Bonus: jika > 2x avg gap, sangat prioritas
                        if actualGap > avgGap*2 {
                                score *= 2.0
                        }
                        ranked = append(ranked, ds{d, score})
                }
                sort.Slice(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })
                // Ambil 5 digit paling overdue per posisi
                for i := 0; i < 5 && i < len(ranked); i++ {
                        candidateDigits[pos] = append(candidateDigits[pos], ranked[i].d)
                }
        }
        pool := combinePositions4D(candidateDigits, 50)
        return diversifyPredictions(pool, 5)
}

// ============================================================
// Method 4: POLA EKOR — analisis pola 2D ekor berulang dalam
// window 30 sesi, cari kombinasi yang sering muncul.
// ============================================================
func predictEkorAS(history []Result) []string {
        if len(history) == 0 {
                return generateRandom(4, 5000)
        }

        n := len(history)
        if n > 60 {
                n = 60
        }
        recent := history[:n]

        // Hitung frekuensi 2D ekor dalam window panjang
        ekor2DFreq := map[string]int{}
        for _, r := range recent {
                nomor := r.Nomor
                for len(nomor) < 4 {
                        nomor = "0" + nomor
                }
                ekor2DFreq[nomor[2:]]++
        }

        // Gap 2D ekor: berapa lama tidak muncul
        ekor2DGap := map[string]int{}
        for k, r := range recent {
                nomor := r.Nomor
                for len(nomor) < 4 {
                        nomor = "0" + nomor
                }
                e := nomor[2:]
                if _, ok := ekor2DGap[e]; !ok {
                        ekor2DGap[e] = k
                }
        }

        // Skor 2D ekor: kombinasi frekuensi + overdue
        type e2d struct {
                ekor  string
                score float64
        }
        var ekorScores []e2d
        for e := 0; e <= 99; e++ {
                es := fmt.Sprintf("%02d", e)
                freq := float64(ekorFreq2D(es, recent))
                gap := float64(n + 5)
                if g, ok := ekor2DGap[es]; ok {
                        gap = float64(g)
                }
                avgGap := float64(n) / (freq + 1.0)
                score := (gap-avgGap)/(avgGap+1.0)*2.0 + freq*0.5
                ekorScores = append(ekorScores, e2d{es, score})
        }
        sort.Slice(ekorScores, func(i, j int) bool { return ekorScores[i].score > ekorScores[j].score })

        // Ambil top 5 ekor, pasang dengan digit AS dari frekuensi menengah
        asFreq := [10]float64{}
        for k, r := range recent {
                d := parse4D(r.Nomor)
                w := math.Exp(-float64(k) * 0.04)
                asFreq[d[0]] += w
        }
        type df struct {
                d int
                f float64
        }
        var asRanked []df
        for d := 0; d < 10; d++ {
                asRanked = append(asRanked, df{d, asFreq[d]})
        }
        sort.Slice(asRanked, func(i, j int) bool { return asRanked[i].f > asRanked[j].f })

        seen := map[string]bool{}
        var results []string
        topEkor := ekorScores[:min(len(ekorScores), 5)]
        for _, e := range topEkor {
                for _, a := range asRanked[:min(len(asRanked), 3)] {
                        nomor := fmt.Sprintf("%d%d%s", a.d, (a.d+int(e.ekor[0]-'0'))%10, e.ekor)
                        if len(nomor) > 4 {
                                nomor = nomor[len(nomor)-4:]
                        }
                        for len(nomor) < 4 {
                                nomor = "0" + nomor
                        }
                        if !seen[nomor] {
                                seen[nomor] = true
                                results = append(results, nomor)
                        }
                        if len(results) >= 5 {
                                break
                        }
                }
                if len(results) >= 5 {
                        break
                }
        }
        for len(results) < 5 {
                rnd := generateRandom(1, 5000+len(results))
                if !seen[rnd[0]] {
                        seen[rnd[0]] = true
                        results = append(results, rnd[0])
                }
        }
        return results[:min(5, len(results))]
}

func ekorFreq2D(ekor string, history []Result) int {
        count := 0
        for _, r := range history {
                nomor := r.Nomor
                for len(nomor) < 4 {
                        nomor = "0" + nomor
                }
                if nomor[2:] == ekor {
                        count++
                }
        }
        return count
}

// ============================================================
// Method 6: MATRIX — analisis korelasi antar posisi digit,
// temukan pasangan posisi yang sering muncul bersama.
// ============================================================
func predictMath(history []Result) []string {
        if len(history) == 0 {
                return generateRandom(4, 8888)
        }

        n := len(history)
        if n > 60 {
                n = 60
        }
        recent := history[:n]

        // Kumpulkan frekuensi digit per posisi (window panjang, decay sangat lemah)
        posFreq := [4][10]float64{}
        for k, r := range recent {
                d := parse4D(r.Nomor)
                w := math.Exp(-float64(k) * 0.03)
                for pos := 0; pos < 4; pos++ {
                        posFreq[pos][d[pos]] += w
                }
        }

        // Gap per posisi
        lastSeen := [4][10]int{}
        for pos := 0; pos < 4; pos++ {
                for d := 0; d < 10; d++ {
                        lastSeen[pos][d] = n + 1
                }
        }
        for k, r := range recent {
                d := parse4D(r.Nomor)
                for pos := 0; pos < 4; pos++ {
                        if lastSeen[pos][d[pos]] == n+1 {
                                lastSeen[pos][d[pos]] = k
                        }
                }
        }

        // Untuk setiap posisi, buat skor kombinasi freq + overdue
        candidateDigits := [4][]int{}
        for pos := 0; pos < 4; pos++ {
                type ds struct {
                        d     int
                        score float64
                }
                var ranked []ds
                for d := 0; d < 10; d++ {
                        gapScore := float64(lastSeen[pos][d]) / float64(n+1)
                        // Bobot: 50% frekuensi, 50% overdue → lebih seimbang dari metode lain
                        score := posFreq[pos][d]*0.5 + gapScore*4.0
                        ranked = append(ranked, ds{d, score})
                }
                sort.Slice(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })
                // 4 kandidat per posisi
                for i := 0; i < 4 && i < len(ranked); i++ {
                        candidateDigits[pos] = append(candidateDigits[pos], ranked[i].d)
                }
        }
        pool := combinePositions4D(candidateDigits, 50)
        return diversifyPredictions(pool, 5)
}

// ============================================================
// GABUNGAN: spread coverage — maksimalkan keragaman 2D ekor
// ============================================================
func predictGabungan(history []Result) []string {
        paito := predictPaito(history)
        shio := predictShio(history)
        ai := predictAI(history)
        ekorAS := predictEkorAS(history)
        mathNums := predictMath(history)

        // Hitung konfirmasi setiap nomor (berapa metode yang merekomendasikan)
        confirmCount := map[string]int{}
        firstSeen := map[string]int{}
        counter := 0
        for _, nums := range [][]string{paito, shio, ai, ekorAS, mathNums} {
                for _, n := range nums {
                        confirmCount[n]++
                        if _, ok := firstSeen[n]; !ok {
                                firstSeen[n] = counter
                                counter++
                        }
                }
        }

        // Buat daftar unik, sort: konfirmasi terbanyak dulu
        type item struct {
                n    string
                c, p int
        }
        seen := map[string]bool{}
        var all []item
        for _, nums := range [][]string{paito, shio, ai, ekorAS, mathNums} {
                for _, n := range nums {
                        if !seen[n] {
                                seen[n] = true
                                all = append(all, item{n, confirmCount[n], firstSeen[n]})
                        }
                }
        }
        sort.Slice(all, func(i, j int) bool {
                if all[i].c != all[j].c {
                        return all[i].c > all[j].c
                }
                return all[i].p < all[j].p
        })

        // Fase 1: masukkan nomor yang dikonfirmasi ≥2 metode
        covered2D := map[string]bool{}
        var result []string
        for _, it := range all {
                if it.c >= 2 {
                        result = append(result, it.n)
                        ekor := it.n
                        for len(ekor) < 4 {
                                ekor = "0" + ekor
                        }
                        covered2D[ekor[2:]] = true
                }
        }

        // Fase 2: isi sisa slot dengan prioritas 2D ekor yang belum tercakup
        for _, it := range all {
                if len(result) >= 20 {
                        break
                }
                alreadyIn := false
                for _, r := range result {
                        if r == it.n {
                                alreadyIn = true
                                break
                        }
                }
                if alreadyIn {
                        continue
                }
                n := it.n
                for len(n) < 4 {
                        n = "0" + n
                }
                if !covered2D[n[2:]] {
                        result = append(result, it.n)
                        covered2D[n[2:]] = true
                }
        }

        // Fase 3: isi sisa dengan nomor apapun yang belum masuk
        for _, it := range all {
                if len(result) >= 20 {
                        break
                }
                alreadyIn := false
                for _, r := range result {
                        if r == it.n {
                                alreadyIn = true
                                break
                        }
                }
                if !alreadyIn {
                        result = append(result, it.n)
                }
        }

        if len(result) > 20 {
                result = result[:20]
        }
        return result
}

// ============================================================
// Sub-digit: 3D dan 2D dipecah dari 4D
// ============================================================
func derive3D(nums4D []string) []string {
        seen := map[string]bool{}
        var result []string
        for _, n := range nums4D {
                for len(n) < 4 {
                        n = "0" + n
                }
                sub := n[1:] // 3 digit terakhir
                if !seen[sub] {
                        seen[sub] = true
                        result = append(result, sub)
                }
        }
        return result
}

func derive2D(nums4D []string) []string {
        seen := map[string]bool{}
        var result []string
        for _, n := range nums4D {
                for len(n) < 4 {
                        n = "0" + n
                }
                sub := n[2:] // 2 digit terakhir
                if !seen[sub] {
                        seen[sub] = true
                        result = append(result, sub)
                }
        }
        return result
}

// ============================================================
// Helpers
// ============================================================
func combinePositions4D(candidates [4][]int, limit int) []string {
        for pos := 0; pos < 4; pos++ {
                if len(candidates[pos]) == 0 {
                        candidates[pos] = []int{0}
                }
        }

        seen := map[string]bool{}
        var results []string

        n := fmt.Sprintf("%d%d%d%d",
                candidates[0][0], candidates[1][0], candidates[2][0], candidates[3][0])
        seen[n] = true
        results = append(results, n)

        for variant := 1; len(results) < limit; variant++ {
                found := false
                for mask := 1; mask < 16; mask++ {
                        digits := [4]int{}
                        for pos := 0; pos < 4; pos++ {
                                cands := candidates[pos]
                                idx := 0
                                if mask&(1<<pos) != 0 {
                                        idx = variant % len(cands)
                                }
                                digits[pos] = cands[idx]
                        }
                        s := fmt.Sprintf("%d%d%d%d", digits[0], digits[1], digits[2], digits[3])
                        if !seen[s] {
                                seen[s] = true
                                results = append(results, s)
                                found = true
                                if len(results) >= limit {
                                        break
                                }
                        }
                }
                if !found || variant > 50 {
                        break
                }
        }
        return results
}

func generateRandom(count, seed int) []string {
        r := rand.New(rand.NewSource(int64(seed)))
        seen := map[string]bool{}
        var results []string
        for len(results) < count {
                n := r.Intn(10000)
                s := fmt.Sprintf("%04d", n)
                if !seen[s] {
                        seen[s] = true
                        results = append(results, s)
                }
        }
        return results
}

func min(a, b int) int {
        if a < b {
                return a
        }
        return b
}

// ============================================================
// BB CAMPURAN — BBFS dari digit terbaik semua metode
// ============================================================

type BBFSNumber struct {
        Nomor string  `json:"nomor"`
        Score float64 `json:"score"`
        Shio  string  `json:"shio"`
        Warna string  `json:"warna"`
        ConfA int     `json:"conf_a"`
}

type BBFSResult struct {
        BBDigits  []int        `json:"bb_digits"`
        DigitFreq [10]int      `json:"digit_freq"`
        Total4D   int          `json:"total_4d"`
        Total3D   int          `json:"total_3d"`
        Total2D   int          `json:"total_2d"`
        Top4D     []BBFSNumber `json:"top_4d"`
        Top3D     []BBFSNumber `json:"top_3d"`
        All2D     []BBFSNumber `json:"all_2d"`
        NDigits   int          `json:"n_digits"`
}

func predictBBFS(history []Result, nDigits int) BBFSResult {
        if nDigits < 4 {
                nDigits = 4
        }
        if nDigits > 7 {
                nDigits = 7
        }
        if len(history) == 0 {
                return BBFSResult{NDigits: nDigits}
        }

        // Step 1: Kumpulkan semua prediksi dari semua metode
        paito := predictPaito(history)
        shio := predictShio(history)
        ai := predictAI(history)
        ekorAS := predictEkorAS(history)
        mathNums := predictMath(history)

        allPreds := []string{}
        for _, s := range [][]string{paito, shio, ai, ekorAS, mathNums} {
                allPreds = append(allPreds, s...)
        }

        // Step 2: Hitung frekuensi setiap digit (25 nomor × 4 digit = 100 digit)
        var digitFreq [10]int
        for _, n := range allPreds {
                d := parse4D(n)
                for _, dig := range d {
                        digitFreq[dig]++
                }
        }

        // Step 3: Pilih top nDigits berdasarkan frekuensi
        type digF struct {
                d, f int
        }
        var ranked []digF
        for d := 0; d < 10; d++ {
                ranked = append(ranked, digF{d, digitFreq[d]})
        }
        sort.Slice(ranked, func(i, j int) bool {
                if ranked[i].f != ranked[j].f {
                        return ranked[i].f > ranked[j].f
                }
                return ranked[i].d < ranked[j].d
        })
        bbDigits := make([]int, nDigits)
        for i := 0; i < nDigits; i++ {
                bbDigits[i] = ranked[i].d
        }

        // Build lookup: exact pred count per nomor/3D/2D
        exactPred4D := map[string]int{}
        pred3DCount := map[string]int{}
        pred2DCount := map[string]int{}
        for _, n := range allPreds {
                p := n
                for len(p) < 4 {
                        p = "0" + p
                }
                exactPred4D[p]++
                pred3DCount[p[1:]]++
                pred2DCount[p[2:]]++
        }

        // ── 4D PERMUTATIONS: P(nDigits, 4) ──────────────────────
        type scored struct {
                n     string
                score float64
                confA int
        }
        var perms4D []scored

        for i := 0; i < nDigits; i++ {
                for j := 0; j < nDigits; j++ {
                        if j == i {
                                continue
                        }
                        for k := 0; k < nDigits; k++ {
                                if k == i || k == j {
                                        continue
                                }
                                for l := 0; l < nDigits; l++ {
                                        if l == i || l == j || l == k {
                                                continue
                                        }
                                        n := fmt.Sprintf("%d%d%d%d", bbDigits[i], bbDigits[j], bbDigits[k], bbDigits[l])
                                        d := parse4D(n)

                                        // Faktor A: nomor persis sama dengan prediksi metode (+3 per metode)
                                        confA := exactPred4D[n]
                                        scoreA := float64(confA) * 3.0

                                        // Faktor B: digit di posisi yang sama dengan prediksi (+0.5 per posisi cocok)
                                        scoreB := 0.0
                                        for _, pred := range allPreds {
                                                pd := parse4D(pred)
                                                for pos := 0; pos < 4; pos++ {
                                                        if d[pos] == pd[pos] {
                                                                scoreB += 0.5
                                                        }
                                                }
                                        }

                                        // Faktor C: digit frequency bonus
                                        scoreC := 0.0
                                        for _, dig := range d {
                                                scoreC += float64(digitFreq[dig]) * 0.2
                                        }

                                        perms4D = append(perms4D, scored{n, scoreA + scoreB + scoreC, confA})
                                }
                        }
                }
        }

        sort.Slice(perms4D, func(i, j int) bool {
                if perms4D[i].score != perms4D[j].score {
                        return perms4D[i].score > perms4D[j].score
                }
                return perms4D[i].n < perms4D[j].n
        })

        top4Dlen := 20
        if len(perms4D) < top4Dlen {
                top4Dlen = len(perms4D)
        }
        top4D := make([]BBFSNumber, top4Dlen)
        for i := 0; i < top4Dlen; i++ {
                s := perms4D[i]
                top4D[i] = BBFSNumber{
                        Nomor: s.n,
                        Score: math.Round(s.score*10) / 10,
                        Shio:  shioOf(s.n),
                        Warna: colorCode4D(s.n),
                        ConfA: s.confA,
                }
        }

        // ── 3D PERMUTATIONS: P(nDigits, 3) ──────────────────────
        type s3d struct {
                n     string
                score float64
                confA int
        }
        seen3D := map[string]bool{}
        var perms3D []s3d

        for i := 0; i < nDigits; i++ {
                for j := 0; j < nDigits; j++ {
                        if j == i {
                                continue
                        }
                        for k := 0; k < nDigits; k++ {
                                if k == i || k == j {
                                        continue
                                }
                                n3 := fmt.Sprintf("%d%d%d", bbDigits[i], bbDigits[j], bbDigits[k])
                                if seen3D[n3] {
                                        continue
                                }
                                seen3D[n3] = true

                                confA := pred3DCount[n3]
                                score := float64(confA) * 3.0
                                score += float64(pred2DCount[n3[1:]]) * 1.5
                                for _, c := range n3 {
                                        dig := int(c - '0')
                                        score += float64(digitFreq[dig]) * 0.15
                                }
                                perms3D = append(perms3D, s3d{n3, score, confA})
                        }
                }
        }
        sort.Slice(perms3D, func(i, j int) bool { return perms3D[i].score > perms3D[j].score })

        top3Dlen := 15
        if len(perms3D) < top3Dlen {
                top3Dlen = len(perms3D)
        }
        top3D := make([]BBFSNumber, top3Dlen)
        for i := 0; i < top3Dlen; i++ {
                top3D[i] = BBFSNumber{Nomor: perms3D[i].n, Score: math.Round(perms3D[i].score*10) / 10, ConfA: perms3D[i].confA}
        }

        // ── 2D PERMUTATIONS: P(nDigits, 2) ──────────────────────
        type s2d struct {
                n     string
                score float64
                confA int
        }
        seen2D := map[string]bool{}
        var perms2D []s2d

        for i := 0; i < nDigits; i++ {
                for j := 0; j < nDigits; j++ {
                        if j == i {
                                continue
                        }
                        n2 := fmt.Sprintf("%d%d", bbDigits[i], bbDigits[j])
                        if seen2D[n2] {
                                continue
                        }
                        seen2D[n2] = true

                        confA := pred2DCount[n2]
                        score := float64(confA) * 3.0
                        d1 := int(n2[0] - '0')
                        d2 := int(n2[1] - '0')
                        score += float64(digitFreq[d1]+digitFreq[d2]) * 0.2
                        perms2D = append(perms2D, s2d{n2, score, confA})
                }
        }
        sort.Slice(perms2D, func(i, j int) bool { return perms2D[i].score > perms2D[j].score })

        all2D := make([]BBFSNumber, len(perms2D))
        for i, s := range perms2D {
                all2D[i] = BBFSNumber{Nomor: s.n, Score: math.Round(s.score*10) / 10, ConfA: s.confA}
        }

        return BBFSResult{
                BBDigits:  bbDigits,
                DigitFreq: digitFreq,
                Total4D:   len(perms4D),
                Total3D:   len(perms3D),
                Total2D:   len(perms2D),
                Top4D:     top4D,
                Top3D:     top3D,
                All2D:     all2D,
                NDigits:   nDigits,
        }
}

// AnalyzePaito untuk tab Paito
func AnalyzePaito(history []Result, limit int) []map[string]interface{} {
        var result []map[string]interface{}
        for i, r := range history {
                if i >= limit {
                        break
                }
                d := parse4D(r.Nomor)
                colors := []string{}
                for _, digit := range d {
                        colors = append(colors, warnaDigit(digit))
                }
                result = append(result, map[string]interface{}{
                        "nomor":   r.Nomor,
                        "tanggal": r.Tanggal,
                        "sesi":    r.Sesi,
                        "colors":  strings.Join(colors, ""),
                        "shio":    shioOf(r.Nomor),
                })
        }
        return result
}
