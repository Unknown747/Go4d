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

func pad5(n int) string {
        return fmt.Sprintf("%05d", n%100000)
}

func parse5D(s string) [5]int {
        s = strings.TrimSpace(s)
        for len(s) < 5 {
                s = "0" + s
        }
        var d [5]int
        for i := 0; i < 5; i++ {
                d[i] = int(s[i] - '0')
        }
        return d
}

// ============================================================
// Method 1: PAITO — improved color streak + wider recency boost
// ============================================================

func predictPaito(history []Result) []string {
        if len(history) == 0 {
                return generateRandom(4, 0)
        }

        candidateDigits := [5][]int{}

        for pos := 0; pos < 5; pos++ {
                pattern := []string{}
                digits := []int{}
                for _, r := range history {
                        d := parse5D(r.Nomor)
                        digits = append(digits, d[pos])
                        pattern = append(pattern, warnaDigit(d[pos]))
                }

                // Detect color streak (up to 5)
                lastColor := ""
                sameCount := 0
                if len(pattern) > 0 {
                        lastColor = pattern[0]
                        for _, c := range pattern {
                                if c == lastColor {
                                        sameCount++
                                } else {
                                        break
                                }
                        }
                }

                predictColor := lastColor
                if sameCount >= 3 {
                        if lastColor == "M" {
                                predictColor = "H"
                        } else {
                                predictColor = "M"
                        }
                }

                // Frequency with recency weighting
                freq := map[int]int{}
                for i, d := range digits {
                        weight := len(digits) - i
                        if warnaDigit(d) == predictColor {
                                freq[d] += weight
                        }
                }

                // Boost most recent appearances (top 3 draws)
                if len(digits) > 0 {
                        freq[digits[0]] += 12
                }
                if len(digits) > 1 {
                        freq[digits[1]] += 6
                }
                if len(digits) > 2 {
                        freq[digits[2]] += 3
                }

                type dfreq struct {
                        d int
                        f int
                }
                var df []dfreq
                for k, v := range freq {
                        df = append(df, dfreq{k, v})
                }
                sort.Slice(df, func(i, j int) bool { return df[i].f > df[j].f })

                maxCandidates := 3
                for i := 0; i < len(df) && i < maxCandidates; i++ {
                        candidateDigits[pos] = append(candidateDigits[pos], df[i].d)
                }

                if len(candidateDigits[pos]) < 2 {
                        if predictColor == "M" {
                                candidateDigits[pos] = []int{1, 3, 5, 7, 9}[:2]
                        } else {
                                candidateDigits[pos] = []int{0, 2, 4, 6, 8}[:2]
                        }
                }
        }

        return combinePositions(candidateDigits, 4)
}

// ============================================================
// Method 2: SHIO — zodiac pattern analysis (unchanged, solid)
// ============================================================

func predictShio(history []Result) []string {
        if len(history) == 0 {
                return generateRandom(4, 1000)
        }

        shioFreq := map[string]float64{}
        shioLast := map[string]int{}

        for i, r := range history {
                s := shioOf(r.Nomor)
                weight := 1.0 / float64(i+1)
                shioFreq[s] += weight
                if _, seen := shioLast[s]; !seen {
                        shioLast[s] = i
                }
        }

        dueScore := map[string]float64{}
        for _, name := range shioNames {
                last, seen := shioLast[name]
                if !seen {
                        last = len(history) + 5
                }
                dueScore[name] = float64(last) - shioFreq[name]*2
        }

        type shioScore struct {
                name  string
                score float64
        }
        var scores []shioScore
        for _, name := range shioNames {
                scores = append(scores, shioScore{name, dueScore[name]})
        }
        sort.Slice(scores, func(i, j int) bool {
                return scores[i].score > scores[j].score
        })

        predictedShio := []string{}
        for i := 0; i < 2 && i < len(scores); i++ {
                predictedShio = append(predictedShio, scores[i].name)
        }

        var results []string
        seen := map[string]bool{}

        for _, shioName := range predictedShio {
                shioIdx := 0
                for i, s := range shioNames {
                        if s == shioName {
                                shioIdx = i
                                break
                        }
                }

                var last2Candidates []int
                for n := 0; n <= 99; n++ {
                        if n%12 == shioIdx {
                                last2Candidates = append(last2Candidates, n)
                        }
                }

                posFreq := [3]map[int]int{}
                for i := 0; i < 3; i++ {
                        posFreq[i] = map[int]int{}
                }
                for k, r := range history {
                        if k >= 15 {
                                break
                        }
                        d := parse5D(r.Nomor)
                        weight := 15 - k
                        for i := 0; i < 3; i++ {
                                posFreq[i][d[i]] += weight
                        }
                }

                topDigits := [3]int{}
                for i := 0; i < 3; i++ {
                        bestD, bestF := 0, -1
                        for d, f := range posFreq[i] {
                                if f > bestF {
                                        bestF = f
                                        bestD = d
                                }
                        }
                        topDigits[i] = bestD
                }

                for _, l2 := range last2Candidates[:min(len(last2Candidates), 4)] {
                        n := topDigits[0]*10000 + topDigits[1]*1000 + topDigits[2]*100 + l2
                        s := pad5(n)
                        if !seen[s] {
                                seen[s] = true
                                results = append(results, s)
                        }
                        if len(results) >= 4 {
                                break
                        }
                }
                if len(results) >= 4 {
                        break
                }
        }

        for len(results) < 4 {
                rnd := generateRandom(1, 2000+len(results))
                if !seen[rnd[0]] {
                        seen[rnd[0]] = true
                        results = append(results, rnd[0])
                }
        }

        return results[:min(4, len(results))]
}

// ============================================================
// Method 3: AI — Frequency + gap + trend + pair/transition
// ============================================================

func predictAI(history []Result) []string {
        if len(history) == 0 {
                return generateRandom(4, 3000)
        }

        n := len(history)
        if n > 30 {
                n = 30
        }
        recent := history[:n]

        // 1. Frequency score with exponential recency weighting
        freqScore := [5][10]float64{}
        for k, r := range recent {
                d := parse5D(r.Nomor)
                weight := math.Exp(-float64(k) * 0.1)
                for pos := 0; pos < 5; pos++ {
                        freqScore[pos][d[pos]] += weight
                }
        }

        // 2. Gap score: digits not seen recently score higher
        lastSeen := [5][10]int{}
        for pos := 0; pos < 5; pos++ {
                for d := 0; d < 10; d++ {
                        lastSeen[pos][d] = n + 10
                }
        }
        for k, r := range recent {
                d := parse5D(r.Nomor)
                for pos := 0; pos < 5; pos++ {
                        if lastSeen[pos][d[pos]] == n+10 {
                                lastSeen[pos][d[pos]] = k
                        }
                }
        }

        // 3. Trend: detect direction per position over last 3 draws
        trendBonus := [5][10]float64{}
        if len(recent) >= 3 {
                for pos := 0; pos < 5; pos++ {
                        d0 := parse5D(recent[0].Nomor)[pos]
                        d1 := parse5D(recent[1].Nomor)[pos]
                        d2 := parse5D(recent[2].Nomor)[pos]

                        if d1 > d2 && d0 > d1 {
                                for d := d0; d <= 9; d++ {
                                        trendBonus[pos][d] += 0.5
                                }
                        } else if d1 < d2 && d0 < d1 {
                                for d := 0; d <= d0; d++ {
                                        trendBonus[pos][d] += 0.5
                                }
                        } else {
                                predicted := (d0 + d1) / 2
                                trendBonus[pos][predicted] += 0.3
                                trendBonus[pos][(predicted+5)%10] += 0.2
                        }
                }
        }

        // 4. NEW: Transition/pair analysis
        // For each position: given the digit that appeared LAST draw,
        // what digit tends to follow it in subsequent draws?
        transScore := [5][10]float64{}
        if len(recent) >= 2 {
                lastDraw := parse5D(recent[0].Nomor)
                for k := 1; k < len(recent); k++ {
                        prev := parse5D(recent[k].Nomor)
                        curr := parse5D(recent[k-1].Nomor)
                        weight := math.Exp(-float64(k) * 0.2)
                        for pos := 0; pos < 5; pos++ {
                                if prev[pos] == lastDraw[pos] {
                                        transScore[pos][curr[pos]] += weight
                                }
                        }
                }
        }

        // 5. NEW: Column pair analysis (digits that frequently appear together
        // in adjacent positions within the same draw)
        pairBonus := [5][10]float64{}
        if len(recent) >= 5 {
                // Build adjacency frequency: how often does digit A at pos P
                // co-occur with digit B at pos P+1?
                // Use this to boost digits that have strong left-neighbor support
                adjFreq := [4][10][10]float64{}
                for k, r := range recent {
                        d := parse5D(r.Nomor)
                        w := math.Exp(-float64(k) * 0.1)
                        for pos := 0; pos < 4; pos++ {
                                adjFreq[pos][d[pos]][d[pos+1]] += w
                        }
                }
                if len(recent) > 0 {
                        lastD := parse5D(recent[0].Nomor)
                        for pos := 1; pos < 5; pos++ {
                                leftDigit := lastD[pos-1]
                                for d := 0; d < 10; d++ {
                                        pairBonus[pos][d] += adjFreq[pos-1][leftDigit][d] * 0.3
                                }
                        }
                }
        }

        // Combine all scores
        finalScore := [5][10]float64{}
        for pos := 0; pos < 5; pos++ {
                for d := 0; d < 10; d++ {
                        gapScore := float64(lastSeen[pos][d]) / float64(n+10)
                        finalScore[pos][d] = freqScore[pos][d]*0.35 +
                                gapScore*0.25 +
                                trendBonus[pos][d]*0.15 +
                                transScore[pos][d]*0.15 +
                                pairBonus[pos][d]*0.10
                }
        }

        candidateDigits := [5][]int{}
        for pos := 0; pos < 5; pos++ {
                type ds struct {
                        d int
                        s float64
                }
                var ranked []ds
                for d := 0; d < 10; d++ {
                        ranked = append(ranked, ds{d, finalScore[pos][d]})
                }
                sort.Slice(ranked, func(i, j int) bool { return ranked[i].s > ranked[j].s })

                for i := 0; i < 3; i++ {
                        candidateDigits[pos] = append(candidateDigits[pos], ranked[i].d)
                }
        }

        return combinePositions(candidateDigits, 4)
}

// ============================================================
// Method 4 (NEW): AS/Ekor — first & last digit focus
// Identifies hottest AS (digit 1) and Ekor (digit 5) combinations
// ============================================================

func predictEkorAS(history []Result) []string {
        if len(history) == 0 {
                return generateRandom(4, 5000)
        }

        n := len(history)
        if n > 40 {
                n = 40
        }
        recent := history[:n]

        // Score AS (pos 0) and Ekor (pos 4) with hot+cold blended analysis
        asFreq := [10]float64{}
        ekorFreq := [10]float64{}
        asLastSeen := [10]int{}
        ekorLastSeen := [10]int{}
        for i := range asLastSeen {
                asLastSeen[i] = n + 5
                ekorLastSeen[i] = n + 5
        }

        for k, r := range recent {
                d := parse5D(r.Nomor)
                w := math.Exp(-float64(k) * 0.08)
                asFreq[d[0]] += w
                ekorFreq[d[4]] += w
                if asLastSeen[d[0]] == n+5 {
                        asLastSeen[d[0]] = k
                }
                if ekorLastSeen[d[4]] == n+5 {
                        ekorLastSeen[d[4]] = k
                }
        }

        type digScore struct {
                d     int
                score float64
        }
        asScores := make([]digScore, 10)
        ekorScores := make([]digScore, 10)
        for d := 0; d < 10; d++ {
                asGap := float64(asLastSeen[d]) / float64(n+5)
                ekorGap := float64(ekorLastSeen[d]) / float64(n+5)
                asScores[d] = digScore{d, asFreq[d]*0.55 + asGap*0.45}
                ekorScores[d] = digScore{d, ekorFreq[d]*0.55 + ekorGap*0.45}
        }
        sort.Slice(asScores, func(i, j int) bool { return asScores[i].score > asScores[j].score })
        sort.Slice(ekorScores, func(i, j int) bool { return ekorScores[i].score > ekorScores[j].score })

        // Middle positions (1-3): use overall frequency weighted by recency
        midFreq := [3][10]float64{}
        for k, r := range recent {
                d := parse5D(r.Nomor)
                w := math.Exp(-float64(k) * 0.1)
                for pos := 1; pos <= 3; pos++ {
                        midFreq[pos-1][d[pos]] += w
                }
        }

        candidateDigits := [5][]int{}
        for i := 0; i < 3; i++ {
                candidateDigits[0] = append(candidateDigits[0], asScores[i].d)
                candidateDigits[4] = append(candidateDigits[4], ekorScores[i].d)
        }
        for pos := 1; pos <= 3; pos++ {
                type ds struct {
                        d int
                        f float64
                }
                var ranked []ds
                for d := 0; d < 10; d++ {
                        ranked = append(ranked, ds{d, midFreq[pos-1][d]})
                }
                sort.Slice(ranked, func(i, j int) bool { return ranked[i].f > ranked[j].f })
                for i := 0; i < 3; i++ {
                        candidateDigits[pos] = append(candidateDigits[pos], ranked[i].d)
                }
        }

        return combinePositions(candidateDigits, 4)
}

// ============================================================
// Method 5: KOP·KEPALA — fokus digit posisi 1 (Kop) & 3 (Kepala)
// Melengkapi AS/Ekor: dua digit "tengah" yang sering diabaikan
// ============================================================

func predictKopKepala(history []Result) []string {
        if len(history) < 3 {
                return generateRandom(4, 7777)
        }

        n := len(history)
        if n > 40 {
                n = 40
        }
        recent := history[:n]

        // Skor frekuensi per posisi, bobot recency
        posFreq := [5][10]float64{}
        for k, r := range recent {
                d := parse5D(r.Nomor)
                w := math.Exp(-float64(k) * 0.08)
                for pos := 0; pos < 5; pos++ {
                        posFreq[pos][d[pos]] += w
                }
        }

        // Skor gap (due): digit lama tidak muncul → kandidat kuat
        lastSeen := [5][10]int{}
        for pos := 0; pos < 5; pos++ {
                for d := 0; d < 10; d++ {
                        lastSeen[pos][d] = 999
                }
        }
        for k, r := range recent {
                d := parse5D(r.Nomor)
                for pos := 0; pos < 5; pos++ {
                        if lastSeen[pos][d[pos]] == 999 {
                                lastSeen[pos][d[pos]] = k
                        }
                }
        }
        gapScore := [5][10]float64{}
        for pos := 0; pos < 5; pos++ {
                for d := 0; d < 10; d++ {
                        gap := lastSeen[pos][d]
                        if gap > 3 && gap < 999 {
                                gapScore[pos][d] = float64(gap) * 0.18
                        } else if gap == 999 {
                                gapScore[pos][d] = 3.0 // belum pernah muncul → prioritas tinggi
                        }
                }
        }

        // Gabungkan: Kop (pos 1) & Kepala (pos 3) diberi bobot 2.5×
        finalScore := [5][10]float64{}
        for pos := 0; pos < 5; pos++ {
                focusWeight := 1.0
                if pos == 1 || pos == 3 {
                        focusWeight = 2.5
                }
                for d := 0; d < 10; d++ {
                        finalScore[pos][d] = posFreq[pos][d]*focusWeight + gapScore[pos][d]*focusWeight
                }
        }

        candidateDigits := [5][]int{}
        for pos := 0; pos < 5; pos++ {
                type ds struct {
                        d int
                        s float64
                }
                var ranked []ds
                for d := 0; d < 10; d++ {
                        ranked = append(ranked, ds{d, finalScore[pos][d]})
                }
                sort.Slice(ranked, func(i, j int) bool { return ranked[i].s > ranked[j].s })
                for i := 0; i < 3; i++ {
                        candidateDigits[pos] = append(candidateDigits[pos], ranked[i].d)
                }
        }

        return combinePositions(candidateDigits, 4)
}

// ============================================================
// Method 6: MATH — rumus matematika klasik
// Cermin, jumlah digit, delta ±, cross formula
// ============================================================

func predictMath(history []Result) []string {
        if len(history) == 0 {
                return generateRandom(4, 8888)
        }

        last := parse5D(history[0].Nomor)
        seen := map[string]bool{}
        var candidates []string

        add := func(d [5]int) {
                s := fmt.Sprintf("%d%d%d%d%d", d[0], d[1], d[2], d[3], d[4])
                if !seen[s] {
                        seen[s] = true
                        candidates = append(candidates, s)
                }
        }

        // 1. Cermin: balik urutan digit
        add([5]int{last[4], last[3], last[2], last[1], last[0]})

        // 2. Jumlah digit → target ekor baru
        sum := 0
        for _, d := range last {
                sum += d
        }
        sumMod := sum % 10
        add([5]int{last[0], last[1], last[2], last[3], sumMod})
        add([5]int{sumMod, last[1], last[2], last[3], last[4]})

        // 3. Cross AS↔Ekor
        add([5]int{last[4], last[1], last[2], last[3], last[0]})

        // 4. Delta +1 semua posisi
        add([5]int{
                (last[0] + 1) % 10, (last[1] + 1) % 10, (last[2] + 1) % 10,
                (last[3] + 1) % 10, (last[4] + 1) % 10,
        })

        // 5. Delta -1 semua posisi
        add([5]int{
                (last[0] + 9) % 10, (last[1] + 9) % 10, (last[2] + 9) % 10,
                (last[3] + 9) % 10, (last[4] + 9) % 10,
        })

        // 6. Flip +5 (komplemen)
        add([5]int{
                (last[0] + 5) % 10, (last[1] + 5) % 10, (last[2] + 5) % 10,
                (last[3] + 5) % 10, (last[4] + 5) % 10,
        })

        // 7. Jumlah AS+Ekor → digit baru
        asEkorSum := (last[0] + last[4]) % 10
        add([5]int{asEkorSum, last[1], last[2], last[3], asEkorSum})

        // Rank candidates by historical frequency support
        n := len(history)
        if n > 20 {
                n = 20
        }
        recent := history[:n]
        freqScore := [5][10]float64{}
        for k, r := range recent {
                d := parse5D(r.Nomor)
                w := math.Exp(-float64(k) * 0.1)
                for pos := 0; pos < 5; pos++ {
                        freqScore[pos][d[pos]] += w
                }
        }

        type rs struct {
                s     string
                score float64
        }
        var ranked []rs
        for _, r := range candidates {
                d := parse5D(r)
                score := 0.0
                for pos := 0; pos < 5; pos++ {
                        score += freqScore[pos][d[pos]]
                }
                ranked = append(ranked, rs{r, score})
        }
        sort.Slice(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })

        var results []string
        seen2 := map[string]bool{}
        for i := 0; i < 4 && i < len(ranked); i++ {
                if !seen2[ranked[i].s] {
                        seen2[ranked[i].s] = true
                        results = append(results, ranked[i].s)
                }
        }
        for len(results) < 4 {
                rnd := generateRandom(1, 8888+len(results))
                if !seen2[rnd[0]] {
                        seen2[rnd[0]] = true
                        results = append(results, rnd[0])
                }
        }
        return results[:4]
}

// ============================================================
// Method 7: SESICORR — korelasi hasil sesi 1 → sesi 2
// Hanya relevan saat prediksi sesi 2 — sesi 1 sudah ada hasilnya
// ============================================================

func predictSesiCorr(pairs []DayPair, sesi1Nomor string) []string {
        if len(pairs) < 3 || sesi1Nomor == "" {
                return nil
        }

        s1 := parse5D(sesi1Nomor)

        // Per posisi: bobot sesi2[pos] berdasarkan kesamaan dengan sesi1 saat ini
        posScore := [5][10]float64{}

        for k, p := range pairs {
                d1 := parse5D(p.Sesi1)
                d2 := parse5D(p.Sesi2)
                w := math.Exp(-float64(k) * 0.12)

                for pos := 0; pos < 5; pos++ {
                        // Cocok persis → boost tinggi
                        if d1[pos] == s1[pos] {
                                posScore[pos][d2[pos]] += w * 3.0
                        }
                        // Selisih 1 → boost ringan
                        if (d1[pos]+1)%10 == s1[pos] || (d1[pos]+9)%10 == s1[pos] {
                                posScore[pos][d2[pos]] += w * 0.8
                        }
                        // Baseline frekuensi sesi 2
                        posScore[pos][d2[pos]] += w * 0.2
                }
        }

        candidateDigits := [5][]int{}
        for pos := 0; pos < 5; pos++ {
                type ds struct {
                        d int
                        s float64
                }
                var ranked []ds
                for d := 0; d < 10; d++ {
                        ranked = append(ranked, ds{d, posScore[pos][d]})
                }
                sort.Slice(ranked, func(i, j int) bool { return ranked[i].s > ranked[j].s })
                for i := 0; i < 3; i++ {
                        candidateDigits[pos] = append(candidateDigits[pos], ranked[i].d)
                }
        }

        return combinePositions(candidateDigits, 4)
}

// ============================================================
// GABUNGAN: Semua nomor unik dari seluruh metode, diurutkan prioritas
// ============================================================

func predictGabungan(history []Result) []string {
        paito := predictPaito(history)
        shio := predictShio(history)
        ai := predictAI(history)
        ekorAS := predictEkorAS(history)
        mathNums := predictMath(history)
        kopKep := predictKopKepala(history)

        seen := map[string]bool{}
        var all []string

        // Urutan prioritas menentukan ranking, tapi SEMUA nomor unik dimasukkan
        // Paito > AI > Kop·Kep > Rumus > Shio > AS/Ekor
        for _, nums := range [][]string{paito, ai, kopKep, mathNums, shio, ekorAS} {
                for _, n := range nums {
                        if !seen[n] {
                                seen[n] = true
                                all = append(all, n)
                        }
                }
        }

        return all
}

// ============================================================
// Sub-digit predictions: 4D, 3D, 2D
// Based on suffix positions of the 5D number
// ============================================================

func predict4D(history []Result) []string {
        return predictSubD(history, 1, 4, 10)
}

func predict3D(history []Result) []string {
        return predictSubD(history, 2, 3, 10)
}

func predict2D(history []Result) []string {
        return predictSubD(history, 3, 2, 10)
}

// predictSubD generates predictions for a contiguous range of digit positions
// startPos: which position in 5D to start from (0-indexed)
// numDigits: how many digits
// limit: how many results to generate
func predictSubD(history []Result, startPos int, numDigits int, limit int) []string {
        maxVal := 1
        for i := 0; i < numDigits; i++ {
                maxVal *= 10
        }

        if len(history) == 0 {
                r := rand.New(rand.NewSource(int64(startPos*1111 + numDigits)))
                seen := map[string]bool{}
                var results []string
                for len(results) < limit {
                        n := r.Intn(maxVal)
                        s := fmt.Sprintf("%0*d", numDigits, n)
                        if !seen[s] {
                                seen[s] = true
                                results = append(results, s)
                        }
                }
                return results
        }

        n := len(history)
        if n > 40 {
                n = 40
        }
        recent := history[:n]

        candidateDigits := [5][]int{}

        for i := 0; i < numDigits; i++ {
                pos := startPos + i
                freqScore := [10]float64{}
                lastSeen := [10]int{}
                for d := range lastSeen {
                        lastSeen[d] = n + 10
                }
                trendBonus := [10]float64{}

                for k, r := range recent {
                        d := parse5D(r.Nomor)[pos]
                        w := math.Exp(-float64(k) * 0.12)
                        freqScore[d] += w
                        if lastSeen[d] == n+10 {
                                lastSeen[d] = k
                        }
                }

                if len(recent) >= 3 {
                        d0 := parse5D(recent[0].Nomor)[pos]
                        d1 := parse5D(recent[1].Nomor)[pos]
                        d2 := parse5D(recent[2].Nomor)[pos]
                        if d0 > d1 && d1 > d2 {
                                for d := d0; d <= 9; d++ {
                                        trendBonus[d] += 0.4
                                }
                        } else if d0 < d1 && d1 < d2 {
                                for d := 0; d <= d0; d++ {
                                        trendBonus[d] += 0.4
                                }
                        } else {
                                mid := (d0 + d1) / 2
                                trendBonus[mid] += 0.3
                                trendBonus[(mid+5)%10] += 0.2
                        }
                }

                finalScore := [10]float64{}
                for d := 0; d < 10; d++ {
                        gapScore := float64(lastSeen[d]) / float64(n+10)
                        finalScore[d] = freqScore[d]*0.5 + gapScore*0.3 + trendBonus[d]*0.2
                }

                type ds struct {
                        d int
                        s float64
                }
                var ranked []ds
                for d := 0; d < 10; d++ {
                        ranked = append(ranked, ds{d, finalScore[d]})
                }
                sort.Slice(ranked, func(i, j int) bool { return ranked[i].s > ranked[j].s })

                for j := 0; j < 3; j++ {
                        candidateDigits[i] = append(candidateDigits[i], ranked[j].d)
                }
        }

        seen := map[string]bool{}
        var results []string

        // First: top combination
        s := ""
        for i := 0; i < numDigits; i++ {
                s += fmt.Sprintf("%d", candidateDigits[i][0])
        }
        seen[s] = true
        results = append(results, s)

        // Variants by rotating candidates
        for variant := 1; len(results) < limit; variant++ {
                for mask := 1; mask < (1 << numDigits); mask++ {
                        digits := make([]int, numDigits)
                        for i := 0; i < numDigits; i++ {
                                cands := candidateDigits[i]
                                idx := 0
                                if mask&(1<<i) != 0 {
                                        idx = variant % len(cands)
                                }
                                digits[i] = cands[idx]
                        }
                        str := ""
                        for _, d := range digits {
                                str += fmt.Sprintf("%d", d)
                        }
                        if !seen[str] {
                                seen[str] = true
                                results = append(results, str)
                                if len(results) >= limit {
                                        break
                                }
                        }
                }
                if variant > 100 {
                        break
                }
        }

        return results
}

// ============================================================
// Deduplication: remove sub-D numbers already covered by higher-D
// A sub-D number X is "covered" if any higher-D number ends with X (suffix match)
// ============================================================

func dedupBySuffix(higherD []string, candidates []string) []string {
        if len(candidates) == 0 {
                return candidates
        }
        candLen := len(candidates[0])

        suffixSet := map[string]bool{}
        for _, n := range higherD {
                if len(n) >= candLen {
                        suffixSet[n[len(n)-candLen:]] = true
                }
        }

        var result []string
        seen := map[string]bool{}
        for _, c := range candidates {
                if len(c) != candLen {
                        continue
                }
                if !suffixSet[c] && !seen[c] {
                        seen[c] = true
                        result = append(result, c)
                }
        }
        return result
}

// ============================================================
// Helper functions
// ============================================================

func combinePositions(candidates [5][]int, limit int) []string {
        // Guard: ensure no position has empty candidates (prevents div-by-zero)
        for pos := 0; pos < 5; pos++ {
                if len(candidates[pos]) == 0 {
                        candidates[pos] = []int{0}
                }
        }

        seen := map[string]bool{}
        var results []string

        n := fmt.Sprintf("%d%d%d%d%d",
                candidates[0][0], candidates[1][0], candidates[2][0],
                candidates[3][0], candidates[4][0])
        seen[n] = true
        results = append(results, n)

        for variant := 1; len(results) < limit; variant++ {
                found := false
                for mask := 1; mask < 32; mask++ {
                        digits := [5]int{}
                        for pos := 0; pos < 5; pos++ {
                                cands := candidates[pos]
                                idx := 0
                                if mask&(1<<pos) != 0 {
                                        idx = variant % len(cands)
                                }
                                digits[pos] = cands[idx]
                        }
                        s := fmt.Sprintf("%d%d%d%d%d", digits[0], digits[1], digits[2], digits[3], digits[4])
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
                n := r.Intn(100000)
                s := fmt.Sprintf("%05d", n)
                if !seen[s] {
                        seen[s] = true
                        results = append(results, s)
                }
        }
        return results
}

// AnalyzePaito returns paito color analysis for display
func AnalyzePaito(history []Result, limit int) []map[string]interface{} {
        var result []map[string]interface{}
        for i, r := range history {
                if i >= limit {
                        break
                }
                d := parse5D(r.Nomor)
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
