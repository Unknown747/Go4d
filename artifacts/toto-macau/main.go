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
        mux.HandleFunc("/history", handleHistory)
        mux.HandleFunc("/paito", handlePaito)
        mux.HandleFunc("/winrate", handleWinRate)
        mux.HandleFunc("/backtest", handleBacktest)

        log.Printf("Server berjalan di port %s", port)
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

        r1, s1Done := getTodayResult(today, 1)
        r2, s2Done := getTodayResult(today, 2)

        data := map[string]interface{}{
                "today":        today,
                "next_tanggal": nextTanggal,
                "next_sesi":    nextSesi,
                "sesi1_done":   s1Done,
                "sesi2_done":   s2Done,
                "sesi1_result": r1.Nomor,
                "sesi2_result": r2.Nomor,
                "server_time":  nowWIB().Format("02 Jan 2006 15:04 WIB"),
                "jadwal_sesi1": "15:15 WIB",
                "jadwal_sesi2": "21:15 WIB",
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
                if err != nil || sesi < 1 || sesi > 2 {
                        http.Error(w, "Parameter sesi tidak valid (harus 1 atau 2)", http.StatusBadRequest)
                        return
                }
        }

        preds := getLatestPredictions(tanggal, sesi)
        if len(preds) < 10 {
                history := getRecentResults(50)
                generateAndSavePredictions(tanggal, sesi, history)
                preds = getLatestPredictions(tanggal, sesi)
        }

        // Bangun map nomor per metode dari DB
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

        // === GABUNGAN 5D: max 20 nomor, diranking dari jumlah metode yang mengkonfirmasi ===
        // Makin banyak metode yang prediksi nomor tersebut → makin tinggi prioritasnya
        type numScore struct {
                nomor    string
                count    int
                priority int
        }
        confirmCount5D := map[string]int{}
        firstSeen5D := map[string]int{}
        prioCounter := 0
        priorityKeys := []string{"paito", "ai", "kopkep", "math", "shio", "ekoras", "sesicorr"}
        for _, key := range priorityKeys {
                for _, n := range methodNums[key] {
                        confirmCount5D[n]++
                        if _, exists := firstSeen5D[n]; !exists {
                                firstSeen5D[n] = prioCounter
                                prioCounter++
                        }
                }
        }
        seenAll5D := map[string]bool{}
        var scoredList []numScore
        for _, key := range priorityKeys {
                for _, n := range methodNums[key] {
                        if !seenAll5D[n] {
                                seenAll5D[n] = true
                                scoredList = append(scoredList, numScore{n, confirmCount5D[n], firstSeen5D[n]})
                        }
                }
        }
        // Sort: konfirmasi terbanyak dulu, lalu urutan prioritas metode
        sort.Slice(scoredList, func(i, j int) bool {
                if scoredList[i].count != scoredList[j].count {
                        return scoredList[i].count > scoredList[j].count
                }
                return scoredList[i].priority < scoredList[j].priority
        })
        var gabungan5D []string
        for i, s := range scoredList {
                if i >= 20 {
                        break
                }
                gabungan5D = append(gabungan5D, s.nomor)
        }
        methodNums["gabungan"] = gabungan5D

        // === 4D: pecah dari tiap nomor 5D (suffix 4 digit), lalu tambah prediksi 4D khusus ===
        seen4D := map[string]bool{}
        var gabungan4D []string
        for _, n := range gabungan5D {
                if len(n) >= 4 {
                        sub := n[len(n)-4:]
                        if !seen4D[sub] {
                                seen4D[sub] = true
                                gabungan4D = append(gabungan4D, sub)
                        }
                }
        }
        for _, n := range methodNums["4d"] {
                if len(n) == 4 && !seen4D[n] {
                        seen4D[n] = true
                        gabungan4D = append(gabungan4D, n)
                }
        }
        methodNums["4d"] = gabungan4D

        // === 3D: pecah dari tiap nomor 5D (suffix 3 digit), lalu tambah prediksi 3D khusus ===
        seen3D := map[string]bool{}
        var gabungan3D []string
        for _, n := range gabungan5D {
                if len(n) >= 3 {
                        sub := n[len(n)-3:]
                        if !seen3D[sub] {
                                seen3D[sub] = true
                                gabungan3D = append(gabungan3D, sub)
                        }
                }
        }
        for _, n := range methodNums["3d"] {
                if len(n) == 3 && !seen3D[n] {
                        seen3D[n] = true
                        gabungan3D = append(gabungan3D, n)
                }
        }
        methodNums["3d"] = gabungan3D

        // === 2D: pecah dari tiap nomor 5D (suffix 2 digit), lalu tambah prediksi 2D khusus ===
        seen2D := map[string]bool{}
        var gabungan2D []string
        for _, n := range gabungan5D {
                if len(n) >= 2 {
                        sub := n[len(n)-2:]
                        if !seen2D[sub] {
                                seen2D[sub] = true
                                gabungan2D = append(gabungan2D, sub)
                        }
                }
        }
        for _, n := range methodNums["2d"] {
                if len(n) == 2 && !seen2D[n] {
                        seen2D[n] = true
                        gabungan2D = append(gabungan2D, n)
                }
        }
        methodNums["2d"] = gabungan2D

        // Bangun response JSON dengan shio + warna per nomor
        result := map[string]interface{}{
                "tanggal": tanggal,
                "sesi":    sesi,
        }
        for key, numbers := range methodNums {
                var withShio []map[string]string
                for _, n := range numbers {
                        withShio = append(withShio, map[string]string{
                                "nomor": n,
                                "shio":  shioOf(n),
                                "warna": colorCode(n),
                        })
                }
                result[key] = withShio
        }

        jsonResponse(w, result)
}

// POST /results  — simpan hasil baru
// PATCH /results — update periode saja
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
                        http.Error(w, "Failed to update: "+err.Error(), 500)
                        return
                }
                jsonResponse(w, map[string]interface{}{"success": true, "message": "Periode diperbarui"})
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
                        http.Error(w, "Nomor hanya boleh berisi angka 0-9", http.StatusBadRequest)
                        return
                }
        }
        if nomor == "" {
                http.Error(w, "Nomor tidak boleh kosong", http.StatusBadRequest)
                return
        }
        for len(nomor) < 5 {
                nomor = "0" + nomor
        }
        if len(nomor) > 5 {
                nomor = nomor[len(nomor)-5:]
        }

        if body.Tanggal == "" {
                body.Tanggal = todayStr()
        }
        if body.Sesi != 1 && body.Sesi != 2 {
                http.Error(w, "Sesi harus 1 atau 2", http.StatusBadRequest)
                return
        }

        if err := saveResult(body.Periode, body.Tanggal, body.Sesi, nomor); err != nil {
                http.Error(w, "Failed to save: "+err.Error(), 500)
                return
        }

        // Auto-generate predictions for next session (INSERT OR IGNORE — won't overwrite existing)
        nextTanggal, nextSesi := nextSessionInfo()
        history := getRecentResults(50)
        generateAndSavePredictions(nextTanggal, nextSesi, history)

        jsonResponse(w, map[string]interface{}{
                "success":      true,
                "message":      fmt.Sprintf("Hasil sesi %d tanggal %s berhasil disimpan", body.Sesi, body.Tanggal),
                "next_tanggal": nextTanggal,
                "next_sesi":    nextSesi,
        })
}

// GET /history
func handleHistory(w http.ResponseWriter, r *http.Request) {
        history := getAllHistory(50)
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
        jsonResponse(w, map[string]interface{}{
                "paito": paito,
        })
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

func generateAndSavePredictions(tanggal string, sesi int, history []Result) {
        paito := predictPaito(history)
        shioNums := predictShio(history)
        ai := predictAI(history)
        ekorAS := predictEkorAS(history)
        kopKep := predictKopKepala(history)
        mathNums := predictMath(history)
        gabungan := predictGabungan(history)

        // Generate 4D, 3D, 2D then deduplicate against higher-D
        raw4D := predict4D(history)
        raw3D := predict3D(history)
        raw2D := predict2D(history)

        deduped4D := dedupBySuffix(gabungan, raw4D)
        higher4D := append(append([]string{}, gabungan...), deduped4D...)
        deduped3D := dedupBySuffix(higher4D, raw3D)
        higher3D := append(append([]string{}, higher4D...), deduped3D...)
        deduped2D := dedupBySuffix(higher3D, raw2D)

        // INSERT OR IGNORE — predictions set once, never overwritten
        savePredictions(tanggal, sesi, "PAITO", paito)
        savePredictions(tanggal, sesi, "SHIO", shioNums)
        savePredictions(tanggal, sesi, "AI", ai)
        savePredictions(tanggal, sesi, "EKORAS", ekorAS)
        savePredictions(tanggal, sesi, "KOPKEP", kopKep)
        savePredictions(tanggal, sesi, "MATH", mathNums)
        savePredictions(tanggal, sesi, "GABUNGAN", gabungan)
        savePredictions(tanggal, sesi, "4D", deduped4D)
        savePredictions(tanggal, sesi, "3D", deduped3D)
        savePredictions(tanggal, sesi, "2D", deduped2D)

        // Korelasi sesi hanya untuk sesi 2 — butuh hasil sesi 1 hari yang sama
        if sesi == 2 {
                sesi1Result, ok := getTodayResult(tanggal, 1)
                if ok && sesi1Result.Nomor != "" {
                        pairs := getDayPairs(30)
                        corrNums := predictSesiCorr(pairs, sesi1Result.Nomor)
                        if len(corrNums) > 0 {
                                savePredictions(tanggal, sesi, "SESICORR", corrNums)
                        }
                }
        }
}

func colorCode(nomor string) string {
        if len(nomor) == 0 {
                return ""
        }
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
