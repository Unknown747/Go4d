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
// Method 1: PAITO — pola warna M/H per posisi + streak
// ============================================================
func predictPaito(history []Result) []string {
	if len(history) == 0 {
		return generateRandom(4, 0)
	}

	candidateDigits := [4][]int{}

	for pos := 0; pos < 4; pos++ {
		pattern := []string{}
		digits := []int{}
		for _, r := range history {
			d := parse4D(r.Nomor)
			digits = append(digits, d[pos])
			pattern = append(pattern, warnaDigit(d[pos]))
		}

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

		freq := map[int]int{}
		for i, d := range digits {
			weight := len(digits) - i
			if warnaDigit(d) == predictColor {
				freq[d] += weight
			}
		}

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

		for i := 0; i < len(df) && i < 3; i++ {
			candidateDigits[pos] = append(candidateDigits[pos], df[i].d)
		}

		if len(candidateDigits[pos]) < 2 {
			if predictColor == "M" {
				candidateDigits[pos] = []int{1, 3, 5}
			} else {
				candidateDigits[pos] = []int{0, 2, 4}
			}
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
// Method 3: AI — frekuensi + gap + tren + transisi + pair
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

	freqScore := [4][10]float64{}
	for k, r := range recent {
		d := parse4D(r.Nomor)
		weight := math.Exp(-float64(k) * 0.1)
		for pos := 0; pos < 4; pos++ {
			freqScore[pos][d[pos]] += weight
		}
	}

	lastSeen := [4][10]int{}
	for pos := 0; pos < 4; pos++ {
		for d := 0; d < 10; d++ {
			lastSeen[pos][d] = n + 10
		}
	}
	for k, r := range recent {
		d := parse4D(r.Nomor)
		for pos := 0; pos < 4; pos++ {
			if lastSeen[pos][d[pos]] == n+10 {
				lastSeen[pos][d[pos]] = k
			}
		}
	}

	trendBonus := [4][10]float64{}
	if len(recent) >= 3 {
		for pos := 0; pos < 4; pos++ {
			d0 := parse4D(recent[0].Nomor)[pos]
			d1 := parse4D(recent[1].Nomor)[pos]
			d2 := parse4D(recent[2].Nomor)[pos]
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

	transScore := [4][10]float64{}
	if len(recent) >= 2 {
		lastDraw := parse4D(recent[0].Nomor)
		for k := 1; k < len(recent); k++ {
			prev := parse4D(recent[k].Nomor)
			curr := parse4D(recent[k-1].Nomor)
			weight := math.Exp(-float64(k) * 0.2)
			for pos := 0; pos < 4; pos++ {
				if prev[pos] == lastDraw[pos] {
					transScore[pos][curr[pos]] += weight
				}
			}
		}
	}

	pairBonus := [4][10]float64{}
	if len(recent) >= 5 {
		adjFreq := [3][10][10]float64{}
		for k, r := range recent {
			d := parse4D(r.Nomor)
			w := math.Exp(-float64(k) * 0.1)
			for pos := 0; pos < 3; pos++ {
				adjFreq[pos][d[pos]][d[pos+1]] += w
			}
		}
		lastD := parse4D(recent[0].Nomor)
		for pos := 1; pos < 4; pos++ {
			leftDigit := lastD[pos-1]
			for d := 0; d < 10; d++ {
				pairBonus[pos][d] += adjFreq[pos-1][leftDigit][d] * 0.3
			}
		}
	}

	finalScore := [4][10]float64{}
	for pos := 0; pos < 4; pos++ {
		for d := 0; d < 10; d++ {
			gapScore := float64(lastSeen[pos][d]) / float64(n+10)
			finalScore[pos][d] = freqScore[pos][d]*0.35 +
				gapScore*0.25 +
				trendBonus[pos][d]*0.15 +
				transScore[pos][d]*0.15 +
				pairBonus[pos][d]*0.10
		}
	}

	candidateDigits := [4][]int{}
	for pos := 0; pos < 4; pos++ {
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
// Method 5: KOP·KEPALA — fokus posisi tengah (pos 1 & 2)
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

	posFreq := [4][10]float64{}
	for k, r := range recent {
		d := parse4D(r.Nomor)
		w := math.Exp(-float64(k) * 0.08)
		for pos := 0; pos < 4; pos++ {
			posFreq[pos][d[pos]] += w
		}
	}

	lastSeen := [4][10]int{}
	for pos := 0; pos < 4; pos++ {
		for d := 0; d < 10; d++ {
			lastSeen[pos][d] = 999
		}
	}
	for k, r := range recent {
		d := parse4D(r.Nomor)
		for pos := 0; pos < 4; pos++ {
			if lastSeen[pos][d[pos]] == 999 {
				lastSeen[pos][d[pos]] = k
			}
		}
	}

	gapScore := [4][10]float64{}
	for pos := 0; pos < 4; pos++ {
		for d := 0; d < 10; d++ {
			gap := lastSeen[pos][d]
			if gap > 3 && gap < 999 {
				gapScore[pos][d] = float64(gap) * 0.18
			} else if gap == 999 {
				gapScore[pos][d] = 3.0
			}
		}
	}

	finalScore := [4][10]float64{}
	for pos := 0; pos < 4; pos++ {
		focusWeight := 1.0
		if pos == 1 || pos == 2 {
			focusWeight = 2.5
		}
		for d := 0; d < 10; d++ {
			finalScore[pos][d] = posFreq[pos][d]*focusWeight + gapScore[pos][d]*focusWeight
		}
	}

	candidateDigits := [4][]int{}
	for pos := 0; pos < 4; pos++ {
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
// GABUNGAN: semua metode, diranking dari jumlah konfirmasi
// ============================================================
func predictGabungan(history []Result) []string {
	paito := predictPaito(history)
	shio := predictShio(history)
	ai := predictAI(history)
	ekorAS := predictEkorAS(history)
	kopKep := predictKopKepala(history)
	mathNums := predictMath(history)

	confirmCount := map[string]int{}
	firstSeen := map[string]int{}
	counter := 0

	for _, nums := range [][]string{paito, ai, kopKep, mathNums, shio, ekorAS} {
		for _, n := range nums {
			confirmCount[n]++
			if _, exists := firstSeen[n]; !exists {
				firstSeen[n] = counter
				counter++
			}
		}
	}

	type scored struct {
		nomor string
		count int
		prio  int
	}
	seen := map[string]bool{}
	var list []scored
	for _, nums := range [][]string{paito, ai, kopKep, mathNums, shio, ekorAS} {
		for _, n := range nums {
			if !seen[n] {
				seen[n] = true
				list = append(list, scored{n, confirmCount[n], firstSeen[n]})
			}
		}
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].count != list[j].count {
			return list[i].count > list[j].count
		}
		return list[i].prio < list[j].prio
	})

	var result []string
	for i, s := range list {
		if i >= 20 {
			break
		}
		result = append(result, s.nomor)
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
