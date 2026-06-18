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
// Method 1: PAITO — warna dominan per posisi + frekuensi terbaru
// FIX: 5 kandidat per posisi (bukan 3), boost warna lebih lunak,
//      generate 40 kombinasi lalu pilih 5 dengan ekor beragam.
// ============================================================
func predictPaito(history []Result) []string {
        if len(history) == 0 {
                return generateRandom(4, 0)
        }

        n := len(history)
        if n > 50 {
                n = 50
        }
        recent := history[:n]

        candidateDigits := [4][]int{}
        for pos := 0; pos < 4; pos++ {
                checkN := 10
                if checkN > n {
                        checkN = n
                }
                mCount, hCount := 0, 0
                for _, r := range recent[:checkN] {
                        d := parse4D(r.Nomor)
                        if d[pos]%2 == 1 {
                                mCount++
                        } else {
                                hCount++
                        }
                }

                // Streak detection — jika 4+ draw berturut warna sama, prediksi balik
                checkStreak := 6
                if checkStreak > n {
                        checkStreak = n
                }
                lastColor := ""
                streakCount := 0
                for _, r := range recent[:checkStreak] {
                        c := "H"
                        d := parse4D(r.Nomor)
                        if d[pos]%2 == 1 {
                                c = "M"
                        }
                        if lastColor == "" {
                                lastColor = c
                        }
                        if c == lastColor {
                                streakCount++
                        } else {
                                break
                        }
                }

                targetColor := "H"
                if mCount > hCount {
                        targetColor = "M"
                }
                if streakCount >= 4 {
                        if lastColor == "M" {
                                targetColor = "H"
                        } else {
                                targetColor = "M"
                        }
                }

                // Skor digit: frekuensi berbobot eksponensial + boost warna target (lebih lunak)
                freq := [10]float64{}
                for k, r := range recent {
                        d := parse4D(r.Nomor)
                        w := math.Exp(-float64(k) * 0.07)
                        freq[d[pos]] += w
                }

                type ds struct {
                        d int
                        s float64
                }
                var ranked []ds
                for d := 0; d < 10; d++ {
                        warna := "H"
                        if d%2 == 1 {
                                warna = "M"
                        }
                        s := freq[d]
                        if warna == targetColor {
                                s *= 1.4 // boost lebih lunak (sebelumnya 2.2)
                        }
                        ranked = append(ranked, ds{d, s})
                }
                sort.Slice(ranked, func(i, j int) bool { return ranked[i].s > ranked[j].s })
                // 5 kandidat per posisi (sebelumnya 3) → lebih banyak variasi ekor
                for i := 0; i < 5; i++ {
                        candidateDigits[pos] = append(candidateDigits[pos], ranked[i].d)
                }
        }
        pool := combinePositions4D(candidateDigits, 40)
        return diversifyPredictions(pool, 5)
}

// ============================================================
// Method 2: SHIO — zodiak Tionghoa dari 2 digit terakhir
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
                gap := float64(last)
                // Ekstra boost untuk shio yang sangat lama tidak muncul (15+ draw)
                if last > 15 {
                        gap *= 1.6
                } else if last > 10 {
                        gap *= 1.2
                }
                dueScore[name] = gap - shioFreq[name]*1.5
        }

        type shioScore struct {
                name  string
                score float64
        }
        var scores []shioScore
        for _, name := range shioNames {
                scores = append(scores, shioScore{name, dueScore[name]})
        }
        sort.Slice(scores, func(i, j int) bool { return scores[i].score > scores[j].score })

        predictedShio := []string{}
        for i := 0; i < 2 && i < len(scores); i++ {
                predictedShio = append(predictedShio, scores[i].name)
        }

        var results []string
        seen := map[string]bool{}

        posFreq := [2]map[int]int{}
        for i := 0; i < 2; i++ {
                posFreq[i] = map[int]int{}
        }
        for k, r := range history {
                if k >= 15 {
                        break
                }
                d := parse4D(r.Nomor)
                weight := 15 - k
                for i := 0; i < 2; i++ {
                        posFreq[i][d[i]] += weight
                }
        }
        topDigits := [2]int{}
        for i := 0; i < 2; i++ {
                bestD, bestF := 0, -1
                for d, f := range posFreq[i] {
                        if f > bestF {
                                bestF = f
                                bestD = d
                        }
                }
                topDigits[i] = bestD
        }

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

                for _, l2 := range last2Candidates[:min(len(last2Candidates), 4)] {
                        n := topDigits[0]*1000 + topDigits[1]*100 + l2
                        s := pad4(n)
                        if !seen[s] {
                                seen[s] = true
                                results = append(results, s)
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
// Method 3: HOT·COLD — blend digit sering keluar & lama tidak muncul
// FIX: 3 hot + 2 cold per posisi (bukan 2+1), generate 40, diversify.
// ============================================================
func predictAI(history []Result) []string {
        if len(history) == 0 {
                return generateRandom(4, 3000)
        }

        n := len(history)
        if n > 40 {
                n = 40
        }
        recent := history[:n]

        hotN := 15
        if hotN > n {
                hotN = n
        }

        // Hot: frekuensi berbobot eksponensial
        hotFreq := [4][10]float64{}
        for k, r := range recent[:hotN] {
                d := parse4D(r.Nomor)
                w := math.Exp(-float64(k) * 0.10)
                for pos := 0; pos < 4; pos++ {
                        hotFreq[pos][d[pos]] += w
                }
        }

        // Cold: paling lama tidak muncul per posisi
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

        candidateDigits := [4][]int{}
        for pos := 0; pos < 4; pos++ {
                type ds struct {
                        d     int
                        score float64
                }
                var hotR, coldR []ds
                for d := 0; d < 10; d++ {
                        hotR = append(hotR, ds{d, hotFreq[pos][d]})
                        coldR = append(coldR, ds{d, float64(lastSeen[pos][d])})
                }
                sort.Slice(hotR, func(i, j int) bool { return hotR[i].score > hotR[j].score })
                sort.Slice(coldR, func(i, j int) bool { return coldR[i].score > coldR[j].score })

                seen := map[int]bool{}
                // 3 digit hot (paling sering keluar)
                for i := 0; i < 3 && i < len(hotR); i++ {
                        candidateDigits[pos] = append(candidateDigits[pos], hotR[i].d)
                        seen[hotR[i].d] = true
                }
                // 2 digit cold (paling lama tidak muncul)
                coldPicked := 0
                for _, cd := range coldR {
                        if !seen[cd.d] {
                                candidateDigits[pos] = append(candidateDigits[pos], cd.d)
                                seen[cd.d] = true
                                coldPicked++
                                if coldPicked >= 2 {
                                        break
                                }
                        }
                }
        }
        pool := combinePositions4D(candidateDigits, 40)
        return diversifyPredictions(pool, 5)
}

// ============================================================
// Method 4: AS/EKOR — fokus digit AS (pos 0) & Ekor (pos 3)
// FIX: 4 kandidat AS, 5 kandidat ekor (termasuk overdue), 3 mid
//      → generate 40, diversify ekor, pilih 5.
// ============================================================
func predictEkorAS(history []Result) []string {
        if len(history) == 0 {
                return generateRandom(4, 5000)
        }

        n := len(history)
        if n > 50 {
                n = 50
        }
        recent := history[:n]

        // Hitung frekuensi + last-seen untuk AS dan Ekor
        asFreq := [10]float64{}
        ekorFreq := [10]float64{}
        asLastSeen := [10]int{}
        ekorLastSeen := [10]int{}
        for i := range asLastSeen {
                asLastSeen[i] = n + 5
                ekorLastSeen[i] = n + 5
        }
        for k, r := range recent {
                d := parse4D(r.Nomor)
                w := math.Exp(-float64(k) * 0.08)
                asFreq[d[0]] += w
                ekorFreq[d[3]] += w
                if asLastSeen[d[0]] == n+5 {
                        asLastSeen[d[0]] = k
                }
                if ekorLastSeen[d[3]] == n+5 {
                        ekorLastSeen[d[3]] = k
                }
        }

        type digScore struct {
                d     int
                score float64
        }

        // AS score: blend frekuensi + overdue
        asScores := make([]digScore, 10)
        for d := 0; d < 10; d++ {
                asGap := float64(asLastSeen[d]) / float64(n+5)
                asScores[d] = digScore{d, asFreq[d]*0.6 + asGap*0.4}
        }
        sort.Slice(asScores, func(i, j int) bool { return asScores[i].score > asScores[j].score })

        // Ekor score: blend frekuensi + overdue per-digit ekor
        ekorScores := make([]digScore, 10)
        for d := 0; d < 10; d++ {
                ekorGap := float64(ekorLastSeen[d]) / float64(n+5)
                ekorScores[d] = digScore{d, ekorFreq[d]*0.5 + ekorGap*0.5}
        }
        sort.Slice(ekorScores, func(i, j int) bool { return ekorScores[i].score > ekorScores[j].score })

        // Mid posisi 1 & 2
        midFreq := [2][10]float64{}
        for k, r := range recent {
                d := parse4D(r.Nomor)
                w := math.Exp(-float64(k) * 0.09)
                for pos := 1; pos <= 2; pos++ {
                        midFreq[pos-1][d[pos]] += w
                }
        }

        candidateDigits := [4][]int{}
        // 4 kandidat AS
        for i := 0; i < 4; i++ {
                candidateDigits[0] = append(candidateDigits[0], asScores[i].d)
        }
        // 5 kandidat ekor untuk variasi 2D lebih luas
        for i := 0; i < 5; i++ {
                candidateDigits[3] = append(candidateDigits[3], ekorScores[i].d)
        }
        // 3 kandidat mid
        for pos := 1; pos <= 2; pos++ {
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

        pool := combinePositions4D(candidateDigits, 40)
        return diversifyPredictions(pool, 5)
}


// ============================================================
// Method 6: MATH — rumus matematika dari angka terakhir & kedua
// FIX: tambah rumus dari hasil ke-2, skor berdasarkan frekuensi,
//      diversify ekor di akhir.
// ============================================================
func predictMath(history []Result) []string {
        if len(history) == 0 {
                return generateRandom(4, 8888)
        }

        last := parse4D(history[0].Nomor)
        seen := map[string]bool{}
        var candidates []string

        add := func(d [4]int) {
                s := fmt.Sprintf("%d%d%d%d", d[0], d[1], d[2], d[3])
                if !seen[s] {
                        seen[s] = true
                        candidates = append(candidates, s)
                }
        }

        // ── Rumus dari hasil terakhir ──
        // 1. Cermin penuh
        add([4]int{last[3], last[2], last[1], last[0]})
        // 2. Jumlah semua digit → ekor baru
        sum := 0
        for _, d := range last {
                sum += d
        }
        sumMod := sum % 10
        add([4]int{last[0], last[1], last[2], sumMod})
        add([4]int{sumMod, last[1], last[2], last[3]})
        // 3. Cross AS↔Ekor
        add([4]int{last[3], last[1], last[2], last[0]})
        // 4. Delta +1
        add([4]int{(last[0] + 1) % 10, (last[1] + 1) % 10, (last[2] + 1) % 10, (last[3] + 1) % 10})
        // 5. Delta -1
        add([4]int{(last[0] + 9) % 10, (last[1] + 9) % 10, (last[2] + 9) % 10, (last[3] + 9) % 10})
        // 6. Flip +5 (pasangan tengkorak)
        add([4]int{(last[0] + 5) % 10, (last[1] + 5) % 10, (last[2] + 5) % 10, (last[3] + 5) % 10})
        // 7. Delta +2 / -2
        add([4]int{(last[0] + 2) % 10, (last[1] + 2) % 10, (last[2] + 2) % 10, (last[3] + 2) % 10})
        add([4]int{(last[0] + 8) % 10, (last[1] + 8) % 10, (last[2] + 8) % 10, (last[3] + 8) % 10})
        // 8. AS-Ekor swap cermin
        add([4]int{last[3], last[2], last[1], last[0]})
        // 9. Komplemen 9 (9-digit)
        add([4]int{(9 - last[0]) % 10, (9 - last[1]) % 10, (9 - last[2]) % 10, (9 - last[3]) % 10})

        // ── Rumus dari hasil kedua (jika ada) ──
        if len(history) >= 2 {
                prev := parse4D(history[1].Nomor)
                // 10. Rata-rata digit per posisi (dibulatkan ke bawah)
                add([4]int{(last[0] + prev[0]) / 2, (last[1] + prev[1]) / 2, (last[2] + prev[2]) / 2, (last[3] + prev[3]) / 2})
                // 11. Selisih absolut
                abs := func(a, b int) int {
                        if a > b {
                                return a - b
                        }
                        return b - a
                }
                add([4]int{abs(last[0], prev[0]), abs(last[1], prev[1]), abs(last[2], prev[2]), abs(last[3], prev[3])})
                // 12. Last ekor + prev AS sebagai ekor baru
                add([4]int{last[0], last[1], last[2], (last[3] + prev[0]) % 10})
                // 13. Cross: last[0..1] + prev[2..3]
                add([4]int{last[0], last[1], prev[2], prev[3]})
                // 14. Cross: prev[0..1] + last[2..3]
                add([4]int{prev[0], prev[1], last[2], last[3]})
        }

        // Skor berdasarkan frekuensi historis per posisi
        hn := len(history)
        if hn > 30 {
                hn = 30
        }
        recent := history[:hn]
        freqScore := [4][10]float64{}
        for k, r := range recent {
                d := parse4D(r.Nomor)
                w := math.Exp(-float64(k) * 0.09)
                for pos := 0; pos < 4; pos++ {
                        freqScore[pos][d[pos]] += w
                }
        }

        type rs struct {
                s     string
                score float64
        }
        var ranked []rs
        for _, r := range candidates {
                d := parse4D(r)
                score := 0.0
                for pos := 0; pos < 4; pos++ {
                        score += freqScore[pos][d[pos]]
                }
                ranked = append(ranked, rs{r, score})
        }
        sort.Slice(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })

        var pool []string
        for _, r := range ranked {
                pool = append(pool, r.s)
        }
        // Tambah padding jika kurang dari 5
        for len(pool) < 10 {
                rnd := generateRandom(1, 8888+len(pool))
                if !seen[rnd[0]] {
                        seen[rnd[0]] = true
                        pool = append(pool, rnd[0])
                }
        }
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
