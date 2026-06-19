package main

import (
        _ "embed"
        "encoding/json"
        "fmt"
        "log"
        "net/http"
        "os"
        "sort"
        "strconv"
        "strings"
)

//go:embed templates/index.html
var indexHTML string

func main() {
        initDB()

        port := os.Getenv("PORT")
        if port == "" {
                port = "8080"
        }

        mux := http.NewServeMux()
        mux.HandleFunc("/", serveIndex)
        mux.HandleFunc("/status", handleStatus)
        mux.HandleFunc("/predictions", handleGetPredictions)
        mux.HandleFunc("/results", handleResults)
        mux.HandleFunc("/import", handleImport)
        mux.HandleFunc("/history", handleHistory)
        mux.HandleFunc("/paito", handlePaito)
        mux.HandleFunc("/winrate", handleWinRate)
        mux.HandleFunc("/backtest", handleBacktest)
        mux.HandleFunc("/bbfs", handleBBFS)
        mux.HandleFunc("/bbbacktest", handleBBBacktest)
        mux.HandleFunc("/rekomendasi", handleRekomendasi)
        mux.HandleFunc("/regenerate", handleRegenerate)
        mux.HandleFunc("/statistik", handleStatistik)
        mux.HandleFunc("/tune", handleTune)
        mux.HandleFunc("/tunehistory", handleTuneHistory)

        log.Printf("Server Macau 4D berjalan di port %s", port)
        if err := http.ListenAndServe(":"+port, corsMiddleware(mux)); err != nil {
                log.Fatal(err)
        }
}

func corsMiddleware(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Access-Control-Allow-Origin", "*")
                w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
                w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
                if r.Method == "OPTIONS" {
                        w.WriteHeader(http.StatusOK)
                        return
                }
                next.ServeHTTP(w, r)
        })
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(data)
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/" && r.URL.Path != "" {
                http.NotFound(w, r)
                return
        }
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        fmt.Fprint(w, indexHTML)
}

// GET /status
func handleStatus(w http.ResponseWriter, r *http.Request) {
        today := todayStr()
        nextTanggal, nextSesi := nextSessionInfo()
        now := nowWIB()

        // Status tiap sesi hari ini
        sesiStatus := []map[string]interface{}{}
        for _, j := range jadwalSesi {
                res, done := getTodayResult(today, j.sesi)
                entry := map[string]interface{}{
                        "sesi":   j.sesi,
                        "jam":    j.jam,
                        "done":   done,
                        "nomor":  res.Nomor,
                }
                sesiStatus = append(sesiStatus, entry)
        }

        data := map[string]interface{}{
                "today":        today,
                "next_tanggal": nextTanggal,
                "next_sesi":    nextSesi,
                "sesi_status":  sesiStatus,
                "server_time":  now.Format("02 Jan 2006 15:04 WIB"),
        }
        jsonResponse(w, data)
}

// GET /predictions
func handleGetPredictions(w http.ResponseWriter, r *http.Request) {
        tanggal := r.URL.Query().Get("tanggal")
        sesiStr := r.URL.Query().Get("sesi")

        var sesi int
        if tanggal == "" || sesiStr == "" {
                tanggal, sesi = nextSessionInfo()
        } else {
                var err error
                sesi, err = strconv.Atoi(sesiStr)
                if err != nil || sesi < 1 || sesi > 6 {
                        http.Error(w, "Sesi harus 1-6", http.StatusBadRequest)
                        return
                }
        }

        preds := getLatestPredictions(tanggal, sesi)
        if len(preds) == 0 {
                history := getRecentResults(100)
                generateAndSavePredictions(tanggal, sesi, history)
                preds = getLatestPredictions(tanggal, sesi)
        }

        // Bangun map per metode
        methodNums := map[string][]string{}
        for _, p := range preds {
                key := strings.ToLower(p.Metode)
                for _, n := range strings.Split(p.NomorList, ",") {
                        n = strings.TrimSpace(n)
                        if n != "" {
                                methodNums[key] = append(methodNums[key], n)
                        }
                }
        }

        // Gabungan 4D: max 20, ranked by konfirmasi
        type numScore struct {
                nomor string
                count int
                prio  int
        }
        confirmCount := map[string]int{}
        firstSeen := map[string]int{}
        prioCounter := 0
        priorityKeys := []string{"paito", "math", "shio"}
        for _, key := range priorityKeys {
                for _, n := range methodNums[key] {
                        confirmCount[n]++
                        if _, exists := firstSeen[n]; !exists {
                                firstSeen[n] = prioCounter
                                prioCounter++
                        }
                }
        }
        seenAll := map[string]bool{}
        var scoredList []numScore
        for _, key := range priorityKeys {
                for _, n := range methodNums[key] {
                        if !seenAll[n] {
                                seenAll[n] = true
                                scoredList = append(scoredList, numScore{n, confirmCount[n], firstSeen[n]})
                        }
                }
        }
        sort.Slice(scoredList, func(i, j int) bool {
                if scoredList[i].count != scoredList[j].count {
                        return scoredList[i].count > scoredList[j].count
                }
                return scoredList[i].prio < scoredList[j].prio
        })
        var gabungan4D []string
        for i, s := range scoredList {
                if i >= 20 {
                        break
                }
                gabungan4D = append(gabungan4D, s.nomor)
        }
        methodNums["gabungan"] = gabungan4D

        // 3D: pecah dari 4D + tambahan unik
        seen3D := map[string]bool{}
        var list3D []string
        for _, n := range gabungan4D {
                for len(n) < 4 {
                        n = "0" + n
                }
                sub := n[1:]
                if !seen3D[sub] {
                        seen3D[sub] = true
                        list3D = append(list3D, sub)
                }
        }
        methodNums["3d"] = list3D

        // 2D: pecah dari 4D + tambahan unik
        seen2D := map[string]bool{}
        var list2D []string
        for _, n := range gabungan4D {
                for len(n) < 4 {
                        n = "0" + n
                }
                sub := n[2:]
                if !seen2D[sub] {
                        seen2D[sub] = true
                        list2D = append(list2D, sub)
                }
        }
        methodNums["2d"] = list2D

        // Bangun response dengan shio + warna
        result := map[string]interface{}{
                "tanggal": tanggal,
                "sesi":    sesi,
        }
        for key, numbers := range methodNums {
                var withMeta []map[string]string
                for _, n := range numbers {
                        withMeta = append(withMeta, map[string]string{
                                "nomor": n,
                                "shio":  shioOf(n),
                                "warna": colorCode4D(n),
                        })
                }
                result[key] = withMeta
        }

        jsonResponse(w, result)
}

// POST /results
// PATCH /results (update periode)
func handleResults(w http.ResponseWriter, r *http.Request) {
        if r.Method == "PATCH" {
                var body struct {
                        Tanggal string `json:"tanggal"`
                        Sesi    int    `json:"sesi"`
                        Periode int    `json:"periode"`
                }
                if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
                        http.Error(w, "Bad request", 400)
                        return
                }
                if err := updatePeriode(body.Tanggal, body.Sesi, body.Periode); err != nil {
                        http.Error(w, "Failed: "+err.Error(), 500)
                        return
                }
                jsonResponse(w, map[string]interface{}{"success": true})
                return
        }
        if r.Method != "POST" {
                http.Error(w, "Method not allowed", 405)
                return
        }

        var body struct {
                Periode int    `json:"periode"`
                Tanggal string `json:"tanggal"`
                Sesi    int    `json:"sesi"`
                Nomor   string `json:"nomor"`
        }
        if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
                http.Error(w, "Bad request", 400)
                return
        }

        nomor := strings.TrimSpace(body.Nomor)
        for _, c := range nomor {
                if c < '0' || c > '9' {
                        http.Error(w, "Nomor hanya boleh angka 0-9", http.StatusBadRequest)
                        return
                }
        }
        if nomor == "" {
                http.Error(w, "Nomor tidak boleh kosong", http.StatusBadRequest)
                return
        }
        for len(nomor) < 4 {
                nomor = "0" + nomor
        }
        if len(nomor) > 4 {
                nomor = nomor[len(nomor)-4:]
        }

        if body.Tanggal == "" {
                body.Tanggal = todayStr()
        }
        if body.Sesi < 1 || body.Sesi > 6 {
                http.Error(w, "Sesi harus 1-6", http.StatusBadRequest)
                return
        }

        if err := saveResult(body.Periode, body.Tanggal, body.Sesi, nomor); err != nil {
                http.Error(w, "Failed to save: "+err.Error(), 500)
                return
        }

        // Generate prediksi untuk sesi berikutnya
        nextTanggal, nextSesi := nextSessionInfo()
        history := getRecentResults(100)
        generateAndSavePredictions(nextTanggal, nextSesi, history)

        jsonResponse(w, map[string]interface{}{
                "success":      true,
                "message":      fmt.Sprintf("Sesi %d tanggal %s disimpan", body.Sesi, body.Tanggal),
                "next_tanggal": nextTanggal,
                "next_sesi":    nextSesi,
        })
}

// POST /import — import massal hasil dari teks
func handleImport(w http.ResponseWriter, r *http.Request) {
        if r.Method != "POST" {
                http.Error(w, "Method not allowed", 405)
                return
        }

        var body struct {
                Data string `json:"data"` // format: "YYYY-MM-DD,sesi,nomor" per baris
        }
        if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
                http.Error(w, "Bad request", 400)
                return
        }

        lines := strings.Split(body.Data, "\n")
        imported := 0
        skipped := 0
        errors := []string{}

        for _, line := range lines {
                line = strings.TrimSpace(line)
                if line == "" || strings.HasPrefix(line, "#") {
                        continue
                }
                // Support comma or tab or space separator
                line = strings.ReplaceAll(line, "\t", ",")
                parts := strings.FieldsFunc(line, func(r rune) bool { return r == ',' || r == ' ' })
                if len(parts) < 3 {
                        skipped++
                        continue
                }

                tanggal := strings.TrimSpace(parts[0])
                sesiStr := strings.TrimSpace(parts[1])
                nomor := strings.TrimSpace(parts[2])

                sesi, err := strconv.Atoi(sesiStr)
                if err != nil || sesi < 1 || sesi > 6 {
                        errors = append(errors, fmt.Sprintf("Baris '%s': sesi tidak valid", line))
                        skipped++
                        continue
                }

                for len(nomor) < 4 {
                        nomor = "0" + nomor
                }
                if len(nomor) > 4 {
                        nomor = nomor[len(nomor)-4:]
                }

                periode := 0
                if len(parts) >= 4 {
                        periode, _ = strconv.Atoi(strings.TrimSpace(parts[3]))
                }

                if err := saveResult(periode, tanggal, sesi, nomor); err != nil {
                        errors = append(errors, fmt.Sprintf("Baris '%s': %v", line, err))
                        skipped++
                } else {
                        imported++
                }
        }

        // Regenerate prediksi setelah import
        nextTanggal, nextSesi := nextSessionInfo()
        history := getRecentResults(100)
        generateAndSavePredictions(nextTanggal, nextSesi, history)

        jsonResponse(w, map[string]interface{}{
                "success":  true,
                "imported": imported,
                "skipped":  skipped,
                "errors":   errors,
                "message":  fmt.Sprintf("Berhasil import %d data, %d dilewati", imported, skipped),
        })
}

// GET /history
func handleHistory(w http.ResponseWriter, r *http.Request) {
        history := getAllHistory(300)
        wr := calculateWinRate()
        jsonResponse(w, map[string]interface{}{
                "history":  history,
                "win_rate": wr,
        })
}

// GET /paito
func handlePaito(w http.ResponseWriter, r *http.Request) {
        history := getRecentResults(30)
        paito := AnalyzePaito(history, 30)
        jsonResponse(w, map[string]interface{}{"paito": paito})
}

// GET /winrate
func handleWinRate(w http.ResponseWriter, r *http.Request) {
        wr := calculateWinRate()
        jsonResponse(w, wr)
}

// GET /backtest
func handleBacktest(w http.ResponseWriter, r *http.Request) {
        report := runBacktest()
        jsonResponse(w, report)
}

// GET /bbfs — always uses 6 digits
func handleBBFS(w http.ResponseWriter, r *http.Request) {
        history := getRecentResults(100)
        result := predictBBFS(history, 6)
        jsonResponse(w, result)
}

// GET /rekomendasi — irisan BB digit + semua prediksi metode
// Menghasilkan nomor yang dikonfirmasi DUA sumber: algoritma + BB digit set
func handleRekomendasi(w http.ResponseWriter, r *http.Request) {
        history := getRecentResults(100)
        bbResult := predictBBFS(history, 6)
        bbSet := map[int]bool{}
        for _, d := range bbResult.BBDigits {
                bbSet[d] = true
        }

        // Kumpulkan prediksi semua metode
        paito := predictPaito(history)
        shio := predictShio(history)
        math := predictMath(history)
        gabungan := predictGabungan(history)

        // Hitung berapa metode yang mengkonfirmasi tiap nomor
        confirmMap := map[string][]string{}
        for _, n := range paito {
                confirmMap[n] = append(confirmMap[n], "Paito")
        }
        for _, n := range shio {
                confirmMap[n] = append(confirmMap[n], "Shio")
        }
        for _, n := range math {
                confirmMap[n] = append(confirmMap[n], "Matrix")
        }

        // Cek apakah semua digit nomor ada dalam BB set
        allInBB := func(nomor string) bool {
                for len(nomor) < 4 {
                        nomor = "0" + nomor
                }
                for _, ch := range nomor {
                        d := int(ch - '0')
                        if !bbSet[d] {
                                return false
                        }
                }
                return true
        }
        last3InBB := func(nomor string) bool {
                for len(nomor) < 4 {
                        nomor = "0" + nomor
                }
                for _, ch := range nomor[1:] {
                        d := int(ch - '0')
                        if !bbSet[d] {
                                return false
                        }
                }
                return true
        }
        last2InBB := func(nomor string) bool {
                for len(nomor) < 4 {
                        nomor = "0" + nomor
                }
                for _, ch := range nomor[2:] {
                        d := int(ch - '0')
                        if !bbSet[d] {
                                return false
                        }
                }
                return true
        }

        type RekItem struct {
                Nomor    string   `json:"nomor"`
                Methods  []string `json:"methods"`
                Count    int      `json:"count"`
                Shio     string   `json:"shio"`
                Warna    string   `json:"warna"`
                InBB4D   bool     `json:"in_bb_4d"`
                InBB3D   bool     `json:"in_bb_3d"`
        }

        seen := map[string]bool{}
        var focus4D, focus3D []RekItem

        for _, n := range gabungan {
                if seen[n] {
                        continue
                }
                seen[n] = true
                n4 := n
                for len(n4) < 4 {
                        n4 = "0" + n4
                }
                methods := confirmMap[n]
                item := RekItem{
                        Nomor:   n4,
                        Methods: methods,
                        Count:   len(methods),
                        Shio:    shioOf(n4),
                        Warna:   colorCode4D(n4),
                        InBB4D:  allInBB(n4),
                        InBB3D:  last3InBB(n4),
                }
                if item.InBB4D {
                        focus4D = append(focus4D, item)
                } else if item.InBB3D {
                        focus3D = append(focus3D, item)
                }
        }

        // Sort by confirmation count descending
        sort.Slice(focus4D, func(i, j int) bool { return focus4D[i].Count > focus4D[j].Count })
        sort.Slice(focus3D, func(i, j int) bool { return focus3D[i].Count > focus3D[j].Count })

        // 2D Fokus: ekor dari semua prediksi yang kedua digitnya ada di BB set
        seen2D := map[string]bool{}
        type Rek2D struct {
                Nomor   string   `json:"nomor"`
                Methods []string `json:"methods"`
                Count   int      `json:"count"`
        }
        var focus2D []Rek2D
        allPreds := append(append(append(paito, shio...), math...), gabungan...)
        for _, n := range allPreds {
                for len(n) < 4 {
                        n = "0" + n
                }
                ekor := n[2:]
                if seen2D[ekor] {
                        continue
                }
                if last2InBB(n) {
                        seen2D[ekor] = true
                        focus2D = append(focus2D, Rek2D{
                                Nomor: ekor,
                                Count: len(confirmMap[n]),
                        })
                }
        }
        // Sort 2D by frequency in predictions
        ekorFreq := map[string]int{}
        for _, n := range allPreds {
                for len(n) < 4 {
                        n = "0" + n
                }
                ekorFreq[n[2:]]++
        }
        sort.Slice(focus2D, func(i, j int) bool {
                return ekorFreq[focus2D[i].Nomor] > ekorFreq[focus2D[j].Nomor]
        })

        jsonResponse(w, map[string]interface{}{
                "bb_digits":      bbResult.BBDigits,
                "focus_4d":       focus4D,
                "focus_3d":       focus3D,
                "focus_2d":       focus2D,
                "gabungan_total": len(gabungan),
                "strategy":       "Nomor dikonfirmasi ganda: algoritma prediksi + digit BB",
        })
}

// GET /bbbacktest
func handleBBBacktest(w http.ResponseWriter, r *http.Request) {
        report := runBBBacktest()
        jsonResponse(w, report)
}

// POST /regenerate — hapus cache dan generate ulang prediksi sesi berikutnya
func handleRegenerate(w http.ResponseWriter, r *http.Request) {
        if r.Method != "POST" {
                http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
                return
        }
        tanggal, sesi := nextSessionInfo()
        deletePredictions(tanggal, sesi)
        history := getRecentResults(100)
        generateAndSavePredictions(tanggal, sesi, history)
        preds := getLatestPredictions(tanggal, sesi)

        methodNums := map[string][]string{}
        for _, p := range preds {
                key := strings.ToLower(p.Metode)
                for _, n := range strings.Split(p.NomorList, ",") {
                        n = strings.TrimSpace(n)
                        if n != "" {
                                methodNums[key] = append(methodNums[key], n)
                        }
                }
        }

        ekorCheck := map[string]int{}
        for method, nums := range methodNums {
                seen := map[string]bool{}
                for _, n := range nums {
                        for len(n) < 4 {
                                n = "0" + n
                        }
                        seen[n[2:]] = true
                }
                ekorCheck[method] = len(seen)
        }

        jsonResponse(w, map[string]interface{}{
                "ok":          true,
                "tanggal":     tanggal,
                "sesi":        sesi,
                "message":     "Prediksi berhasil di-generate ulang",
                "unique_ekor": ekorCheck,
        })
}

func handleStatistik(w http.ResponseWriter, r *http.Request) {
        all := getRecentResults(300)
        if len(all) == 0 {
                jsonResponse(w, map[string]interface{}{"total": 0})
                return
        }
        total := len(all)
        lastDate := all[0].Tanggal
        firstDate := all[total-1].Tanggal

        // Reverse ke oldest-first untuk urutan mingguan
        for i, j := 0, total-1; i < j; i, j = i+1, j-1 {
                all[i], all[j] = all[j], all[i]
        }

        var posFreq [4][10]int
        var posMax [4]int
        ekor2DFreq := map[string]int{}
        var digitAll [10]int
        type colorPos struct {
                Merah int `json:"merah"`
                Hitam int `json:"hitam"`
        }
        var colorPerPos [4]colorPos

        for _, res := range all {
                n := res.Nomor
                for len(n) < 4 {
                        n = "0" + n
                }
                for pos := 0; pos < 4; pos++ {
                        d := int(n[pos] - '0')
                        posFreq[pos][d]++
                        if posFreq[pos][d] > posMax[pos] {
                                posMax[pos] = posFreq[pos][d]
                        }
                        digitAll[d]++
                        if d%2 == 1 {
                                colorPerPos[pos].Merah++
                        } else {
                                colorPerPos[pos].Hitam++
                        }
                }
                ekor2DFreq[n[2:]]++
        }

        type ekor2DItem struct {
                Ekor  string `json:"ekor"`
                Count int    `json:"count"`
        }
        var topEkor2D []ekor2DItem
        for e, c := range ekor2DFreq {
                topEkor2D = append(topEkor2D, ekor2DItem{e, c})
        }
        sort.Slice(topEkor2D, func(i, j int) bool {
                if topEkor2D[i].Count != topEkor2D[j].Count {
                        return topEkor2D[i].Count > topEkor2D[j].Count
                }
                return topEkor2D[i].Ekor < topEkor2D[j].Ekor
        })
        if len(topEkor2D) > 20 {
                topEkor2D = topEkor2D[:20]
        }

        type digitItem struct {
                Digit int     `json:"digit"`
                Count int     `json:"count"`
                Pct   float64 `json:"pct"`
        }
        allCount := total * 4
        var digitAllItems []digitItem
        for d := 0; d < 10; d++ {
                pct := 0.0
                if allCount > 0 {
                        pct = float64(digitAll[d]) / float64(allCount) * 1000
                        pct = float64(int(pct+0.5)) / 10.0
                }
                digitAllItems = append(digitAllItems, digitItem{d, digitAll[d], pct})
        }

        type weekItem struct {
                Label   string   `json:"label"`
                TopEkor []string `json:"top_ekor"`
                Total   int      `json:"total"`
        }
        weekMap := map[string]map[string]int{}
        weekTotals := map[string]int{}
        weekOrder := []string{}
        for _, res := range all {
                parts := strings.Split(res.Tanggal, "-")
                if len(parts) != 3 {
                        continue
                }
                year, _ := strconv.Atoi(parts[0])
                month, _ := strconv.Atoi(parts[1])
                day, _ := strconv.Atoi(parts[2])
                // Hitung ISO week sederhana
                dayOfYear := 0
                daysInMonth := []int{0, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
                if (year%4 == 0 && year%100 != 0) || year%400 == 0 {
                        daysInMonth[2] = 29
                }
                for m := 1; m < month; m++ {
                        dayOfYear += daysInMonth[m]
                }
                dayOfYear += day
                week := (dayOfYear-1)/7 + 1
                key := fmt.Sprintf("%d-W%02d", year, week)
                label := fmt.Sprintf("W%02d/%d", week, year%100)
                if _, ok := weekMap[key]; !ok {
                        weekMap[key] = map[string]int{}
                        weekOrder = append(weekOrder, key)
                        weekMap[key+"_label"] = map[string]int{} // workaround: simpan label
                        _ = label
                }
                n := res.Nomor
                for len(n) < 4 {
                        n = "0" + n
                }
                weekMap[key][n[2:]]++
                weekTotals[key]++
        }
        // Buat label map terpisah
        weekLabels := map[string]string{}
        for _, res := range all {
                parts := strings.Split(res.Tanggal, "-")
                if len(parts) != 3 {
                        continue
                }
                year, _ := strconv.Atoi(parts[0])
                month, _ := strconv.Atoi(parts[1])
                day, _ := strconv.Atoi(parts[2])
                dayOfYear := 0
                daysInMonth := []int{0, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
                if (year%4 == 0 && year%100 != 0) || year%400 == 0 {
                        daysInMonth[2] = 29
                }
                for m := 1; m < month; m++ {
                        dayOfYear += daysInMonth[m]
                }
                dayOfYear += day
                week := (dayOfYear-1)/7 + 1
                key := fmt.Sprintf("%d-W%02d", year, week)
                months := []string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}
                weekLabels[key] = fmt.Sprintf("%s W%d", months[month-1], ((day-1)/7)+1)
        }
        var weekly []weekItem
        start := 0
        if len(weekOrder) > 8 {
                start = len(weekOrder) - 8
        }
        for _, key := range weekOrder[start:] {
                ekorCount := weekMap[key]
                type kv struct {
                        k string
                        v int
                }
                var kvs []kv
                for k, v := range ekorCount {
                        kvs = append(kvs, kv{k, v})
                }
                sort.Slice(kvs, func(i, j int) bool { return kvs[i].v > kvs[j].v })
                var top []string
                for i := 0; i < 5 && i < len(kvs); i++ {
                        top = append(top, fmt.Sprintf("%s(%d)", kvs[i].k, kvs[i].v))
                }
                label := weekLabels[key]
                weekly = append(weekly, weekItem{label, top, weekTotals[key]})
        }

        posFreqSlice := make([][]int, 4)
        posMaxSlice := make([]int, 4)
        for pos := 0; pos < 4; pos++ {
                posFreqSlice[pos] = posFreq[pos][:]
                posMaxSlice[pos] = posMax[pos]
        }

        jsonResponse(w, map[string]interface{}{
                "total":      total,
                "first_date": firstDate,
                "last_date":  lastDate,
                "pos_freq":   posFreqSlice,
                "pos_max":    posMaxSlice,
                "top_ekor2d": topEkor2D,
                "digit_all":  digitAllItems,
                "weekly":     weekly,
                "color_pos":  colorPerPos,
        })
}

// POST /tune — jalankan grid search, update activePaitoConfig, kembalikan hasil
func handleTune(w http.ResponseWriter, r *http.Request) {
        if r.Method != "POST" {
                http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
                return
        }
        result := tunePaitoConfig()
        jsonResponse(w, result)
}

// GET /tunehistory — kembalikan riwayat tuning terakhir
func handleTuneHistory(w http.ResponseWriter, r *http.Request) {
        history := getTuneHistory(20)
        if history == nil {
                history = []TuneHistoryRow{}
        }
        jsonResponse(w, history)
}

func generateAndSavePredictions(tanggal string, sesi int, history []Result) {
        paito := filterPastResults(predictPaito(history), history)
        shioNums := filterPastResults(predictShio(history), history)
        mathNums := filterPastResults(predictMath(history), history)

        savePredictions(tanggal, sesi, "PAITO", paito)
        savePredictions(tanggal, sesi, "SHIO", shioNums)
        savePredictions(tanggal, sesi, "MATH", mathNums)

        gabungan := filterPastResults(predictGabungan(history), history)
        savePredictions(tanggal, sesi, "GABUNGAN", gabungan)
}
