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
        priorityKeys := []string{"paito", "ai", "math", "shio", "ekoras"}
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

// GET /bbbacktest
func handleBBBacktest(w http.ResponseWriter, r *http.Request) {
        report := runBBBacktest()
        jsonResponse(w, report)
}

func generateAndSavePredictions(tanggal string, sesi int, history []Result) {
        paito := predictPaito(history)
        shioNums := predictShio(history)
        ai := predictAI(history)
        ekorAS := predictEkorAS(history)
        mathNums := predictMath(history)

        savePredictions(tanggal, sesi, "PAITO", paito)
        savePredictions(tanggal, sesi, "SHIO", shioNums)
        savePredictions(tanggal, sesi, "AI", ai)
        savePredictions(tanggal, sesi, "EKORAS", ekorAS)
        savePredictions(tanggal, sesi, "MATH", mathNums)

        gabungan := predictGabungan(history)
        savePredictions(tanggal, sesi, "GABUNGAN", gabungan)
}
