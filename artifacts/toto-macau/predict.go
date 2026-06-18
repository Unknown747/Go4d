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
// Method 1: PAITO — warna dominan per posisi + frekuensi terbaru
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
                // Hitung warna dominan berdasarkan 10 draw terakhir
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

                // Deteksi streak — jika 4+ draw berturut warna sama, prediksi balik
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
                        // Streak panjang → antisipasi balik arah
                        if lastColor == "M" {
                                targetColor = "H"
                        } else {
                                targetColor = "M"
                        }
                }

                // Skor tiap digit berdasarkan frekuensi terbaru + boost warna target
                freq := [10]float64{}
                for k, r := range recent {
                        d := parse4D(r.Nomor)
                        w := math.Exp(-float64(k) * 0.08)
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
                                s *= 2.2 // boost digit warna target
                        }
                        ranked = append(ranked, ds{d, s})
                }
                sort.Slice(ranked, func(i, j int) bool { return ranked[i].s > ranked[j].s })
                for i := 0; i < 3; i++ {
                        candidateDigits[pos] = append(candidateDigits[pos], ranked[i].d)
                }
        }
        return combinePositions4D(candidateDigits, 5)
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

        hotN := 12
        if hotN > n {
                hotN = n
        }

        // Hot: frekuensi dalam hotN draw terakhir (bobot eksponensial)
        hotFreq := [4][10]float64{}
        for k, r := range recent[:hotN] {
                d := parse4D(r.Nomor)
                w := math.Exp(-float64(k) * 0.12)
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
                // 2 digit hot (paling sering keluar)
                for i := 0; i < 2 && i < len(hotR); i++ {
                        candidateDigits[pos] = append(candidateDigits[pos], hotR[i].d)
                        seen[hotR[i].d] = true
                }
                // 1 digit cold (paling lama tidak muncul, bukan hot)
                for _, cd := range coldR {
                        if !seen[cd.d] {
                                candidateDigits[pos] = append(candidateDigits[pos], cd.d)
                                break
                        }
                }
        }
        return combinePositions4D(candidateDigits, 5)
}

// ============================================================
// Method 4: AS/EKOR — fokus digit AS (pos 0) & Ekor (pos 3)
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

        // Boost ekor yang overdue: 2D terakhir yang lama tidak muncul
        ekor2DLast := [100]int{}
        for i := range ekor2DLast {
                ekor2DLast[i] = len(recent) + 1
        }
        for k, r := range recent {
                for len(r.Nomor) < 4 {
                        r.Nomor = "0" + r.Nomor
                }
                last2 := (int(r.Nomor[2]-'0') * 10) + int(r.Nomor[3]-'0')
                if ekor2DLast[last2] == len(recent)+1 {
                        ekor2DLast[last2] = k
                }
        }
        // Digit ekor yang paling overdue → boost ekorScore
        for d := 0; d < 10; d++ {
                maxGap := 0
                for t := d; t < 100; t += 10 {
                        if ekor2DLast[t] > maxGap {
                                maxGap = ekor2DLast[t]
                        }
                }
                if maxGap > 8 {
                        for idx, es := range ekorScores {
                                if es.d == d {
                                        ekorScores[idx].score += float64(maxGap) * 0.15
                                }
                        }
                }
        }
        sort.Slice(ekorScores, func(i, j int) bool { return ekorScores[i].score > ekorScores[j].score })

        midFreq := [2][10]float64{}
        for k, r := range recent {
                d := parse4D(r.Nomor)
                w := math.Exp(-float64(k) * 0.1)
                for pos := 1; pos <= 2; pos++ {
                        midFreq[pos-1][d[pos]] += w
                }
        }

        candidateDigits := [4][]int{}
        for i := 0; i < 3; i++ {
                candidateDigits[0] = append(candidateDigits[0], asScores[i].d)
                candidateDigits[3] = append(candidateDigits[3], ekorScores[i].d)
        }
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

        return combinePositions4D(candidateDigits, 5)
}


// ============================================================
// Method 6: MATH — rumus matematika dari angka terakhir
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

        // 1. Cermin
        add([4]int{last[3], last[2], last[1], last[0]})
        // 2. Jumlah digit → ekor baru
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
        // 6. Flip +5
        add([4]int{(last[0] + 5) % 10, (last[1] + 5) % 10, (last[2] + 5) % 10, (last[3] + 5) % 10})
        // 7. AS+Ekor sum
        asEkorSum := (last[0] + last[3]) % 10
        add([4]int{asEkorSum, last[1], last[2], asEkorSum})

        // Rank by historical frequency
        hn := len(history)
        if hn > 20 {
                hn = 20
        }
        recent := history[:hn]
        freqScore := [4][10]float64{}
        for k, r := range recent {
                d := parse4D(r.Nomor)
                w := math.Exp(-float64(k) * 0.1)
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

        var results []string
        seen2 := map[string]bool{}
        for i := 0; i < 5 && i < len(ranked); i++ {
                if !seen2[ranked[i].s] {
                        seen2[ranked[i].s] = true
                        results = append(results, ranked[i].s)
                }
        }
        for len(results) < 5 {
                rnd := generateRandom(1, 8888+len(results))
                if !seen2[rnd[0]] {
                        seen2[rnd[0]] = true
                        results = append(results, rnd[0])
                }
        }
        return results[:5]
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
