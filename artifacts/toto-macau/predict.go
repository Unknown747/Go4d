package main

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
)

// Shio mapping for last 2 digits (0-99 → shio index 0-11)
var shioNames = []string{
	"Tikus", "Kerbau", "Macan", "Kelinci", "Naga", "Ular",
	"Kuda", "Kambing", "Monyet", "Ayam", "Anjing", "Babi",
}

// shioOf returns shio name for a 5D number string
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

// warna (color) of digit: merah = odd, hitam = even
func warnaDigit(d int) string {
	if d%2 == 1 {
		return "M"
	}
	return "H"
}

// pad5 pads a number to 5 digits
func pad5(n int) string {
	return fmt.Sprintf("%05d", n%100000)
}

// parse5D parses a 5D string into 5 digits
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
// Method 1: PAITO — color pattern analysis per position
// ============================================================

func predictPaito(history []Result) []string {
	if len(history) == 0 {
		return generateRandom(4, 0)
	}

	// For each position (0-4), build color pattern and predict next digit
	candidateDigits := [5][]int{}

	for pos := 0; pos < 5; pos++ {
		// Build the color sequence for this position
		pattern := []string{}
		digits := []int{}
		for _, r := range history {
			d := parse5D(r.Nomor)
			digits = append(digits, d[pos])
			pattern = append(pattern, warnaDigit(d[pos]))
		}

		// Count last pattern: if last 3 are same color → predict opposite color
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

		// Determine predicted color
		predictColor := lastColor
		if sameCount >= 3 {
			if lastColor == "M" {
				predictColor = "H"
			} else {
				predictColor = "M"
			}
		}

		// Count frequency of digits matching the predicted color
		freq := map[int]int{}
		for i, d := range digits {
			weight := len(digits) - i
			if warnaDigit(d) == predictColor {
				freq[d] += weight
			}
		}

		// Also factor in: what digits appeared most recently?
		if len(digits) > 0 {
			freq[digits[0]] += 5
		}

		// Sort by frequency
		type dfreq struct {
			d int
			f int
		}
		var df []dfreq
		for k, v := range freq {
			df = append(df, dfreq{k, v})
		}
		sort.Slice(df, func(i, j int) bool { return df[i].f > df[j].f })

		// Take top 2-3 candidates
		maxCandidates := 3
		for i := 0; i < len(df) && i < maxCandidates; i++ {
			candidateDigits[pos] = append(candidateDigits[pos], df[i].d)
		}

		// Ensure we have at least 2 candidates per position
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
// Method 2: SHIO — zodiac pattern analysis
// ============================================================

func predictShio(history []Result) []string {
	if len(history) == 0 {
		return generateRandom(4, 1000)
	}

	// Count shio frequencies with recency weighting
	shioFreq := map[string]float64{}
	shioLast := map[string]int{} // how many draws since last appearance

	for i, r := range history {
		s := shioOf(r.Nomor)
		weight := 1.0 / float64(i+1)
		shioFreq[s] += weight
		if _, seen := shioLast[s]; !seen {
			shioLast[s] = i
		}
	}

	// Calculate "due" score: shio that hasn't appeared in a while
	dueScore := map[string]float64{}
	for _, name := range shioNames {
		last, seen := shioLast[name]
		if !seen {
			last = len(history) + 5
		}
		dueScore[name] = float64(last) - shioFreq[name]*2
	}

	// Sort shio by due score descending
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

	// Take top 2 predicted shio
	predictedShio := []string{}
	for i := 0; i < 2 && i < len(scores); i++ {
		predictedShio = append(predictedShio, scores[i].name)
	}

	// Generate numbers whose last 2 digits match predicted shio
	var results []string
	seen := map[string]bool{}

	for _, shioName := range predictedShio {
		// Find shio index
		shioIdx := 0
		for i, s := range shioNames {
			if s == shioName {
				shioIdx = i
				break
			}
		}

		// Generate last-2-digit values that match this shio
		var last2Candidates []int
		for n := 0; n <= 99; n++ {
			if n%12 == shioIdx {
				last2Candidates = append(last2Candidates, n)
			}
		}

		// For first 3 positions, use statistical pattern from recent history
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

		// Get top digit for each of first 3 positions
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

		// Combine with shio last 2 digits
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

	// Fill up to 4 if needed
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
// Method 3: AI — Statistical + pattern analysis
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

	// 1. Frequency score per digit per position (weighted by recency)
	freqScore := [5][10]float64{}
	for k, r := range recent {
		d := parse5D(r.Nomor)
		weight := math.Exp(-float64(k) * 0.1)
		for pos := 0; pos < 5; pos++ {
			freqScore[pos][d[pos]] += weight
		}
	}

	// 2. Gap score: digits that haven't appeared recently score higher
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

	// 3. Trend: direction of change in last 3 draws per position
	trendBonus := [5][10]float64{}
	if len(recent) >= 3 {
		for pos := 0; pos < 5; pos++ {
			d0 := parse5D(recent[0].Nomor)[pos]
			d1 := parse5D(recent[1].Nomor)[pos]
			d2 := parse5D(recent[2].Nomor)[pos]

			// Detect trend direction
			if d1 > d2 && d0 > d1 {
				// Ascending: predict higher digits
				for d := d0; d <= 9; d++ {
					trendBonus[pos][d] += 0.5
				}
			} else if d1 < d2 && d0 < d1 {
				// Descending: predict lower digits
				for d := 0; d <= d0; d++ {
					trendBonus[pos][d] += 0.5
				}
			} else {
				// Oscillating: predict based on gap
				predicted := (d0 + d1) / 2
				trendBonus[pos][predicted] += 0.3
				trendBonus[pos][(predicted+5)%10] += 0.2
			}
		}
	}

	// Combine all scores
	finalScore := [5][10]float64{}
	for pos := 0; pos < 5; pos++ {
		for d := 0; d < 10; d++ {
			gapScore := float64(lastSeen[pos][d]) / float64(n+10)
			finalScore[pos][d] = freqScore[pos][d]*0.5 + gapScore*0.3 + trendBonus[pos][d]*0.2
		}
	}

	// Pick top 3 digits per position
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
// GABUNGAN: Combine all methods → exactly 10 numbers
// ============================================================

func predictGabungan(history []Result) []string {
	paito := predictPaito(history)
	shio := predictShio(history)
	ai := predictAI(history)

	// Merge all candidates
	seen := map[string]bool{}
	var all []string

	// Add paito results first (weighted)
	for _, n := range paito {
		if !seen[n] {
			seen[n] = true
			all = append(all, n)
		}
	}
	for _, n := range ai {
		if !seen[n] {
			seen[n] = true
			all = append(all, n)
		}
	}
	for _, n := range shio {
		if !seen[n] {
			seen[n] = true
			all = append(all, n)
		}
	}

	// Fill to exactly 10 with pattern-based extras
	seed := 9999
	if len(history) > 0 {
		d := parse5D(history[0].Nomor)
		seed = d[0]*10000 + d[1]*1000 + d[2]*100 + d[3]*10 + d[4]
	}

	extras := generateRandom(20, seed+len(all))
	for _, e := range extras {
		if len(all) >= 10 {
			break
		}
		if !seen[e] {
			seen[e] = true
			all = append(all, e)
		}
	}

	if len(all) > 10 {
		return all[:10]
	}
	return all
}

// ============================================================
// Helper functions
// ============================================================

// combinePositions builds 5D numbers from position candidates
func combinePositions(candidates [5][]int, limit int) []string {
	seen := map[string]bool{}
	var results []string

	// First pass: take the first candidate from each position
	n := fmt.Sprintf("%d%d%d%d%d",
		candidates[0][0], candidates[1][0], candidates[2][0],
		candidates[3][0], candidates[4][0])
	seen[n] = true
	results = append(results, n)

	// Generate variations by rotating through candidates
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

// generateRandom generates random 5D numbers with a seed for consistency
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetShioList returns shio info for a list of numbers
func GetShioList(numbers []string) []map[string]string {
	var result []map[string]string
	for _, n := range numbers {
		result = append(result, map[string]string{
			"nomor": n,
			"shio":  shioOf(n),
		})
	}
	return result
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
