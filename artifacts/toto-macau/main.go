package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
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
	mux.HandleFunc("/results", handlePostResult)
	mux.HandleFunc("/history", handleHistory)
	mux.HandleFunc("/paito", handlePaito)
	mux.HandleFunc("/winrate", handleWinRate)

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
		"server_time":  time.Now().Format("02 Jan 2006 15:04"),
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
		sesi, _ = strconv.Atoi(sesiStr)
	}

	preds := getLatestPredictions(tanggal, sesi)
	if len(preds) == 0 {
		history := getRecentResults(50)
		generateAndSavePredictions(tanggal, sesi, history)
		preds = getLatestPredictions(tanggal, sesi)
	}

	result := map[string]interface{}{
		"tanggal": tanggal,
		"sesi":    sesi,
	}

	for _, p := range preds {
		numbers := strings.Split(p.NomorList, ",")
		var withShio []map[string]string
		for _, n := range numbers {
			n = strings.TrimSpace(n)
			if n == "" {
				continue
			}
			withShio = append(withShio, map[string]string{
				"nomor": n,
				"shio":  shioOf(n),
				"warna": colorCode(n),
			})
		}
		result[strings.ToLower(p.Metode)] = withShio
	}

	jsonResponse(w, result)
}

// POST /results
func handlePostResult(w http.ResponseWriter, r *http.Request) {
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
	for len(nomor) < 5 {
		nomor = "0" + nomor
	}
	if len(nomor) > 5 {
		nomor = nomor[len(nomor)-5:]
	}

	if body.Tanggal == "" {
		body.Tanggal = todayStr()
	}
	if body.Sesi == 0 {
		body.Sesi = 1
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

func generateAndSavePredictions(tanggal string, sesi int, history []Result) {
	paito := predictPaito(history)
	shioNums := predictShio(history)
	ai := predictAI(history)
	gabungan := predictGabungan(history)

	// INSERT OR IGNORE — predictions set once, never overwritten
	savePredictions(tanggal, sesi, "PAITO", paito)
	savePredictions(tanggal, sesi, "SHIO", shioNums)
	savePredictions(tanggal, sesi, "AI", ai)
	savePredictions(tanggal, sesi, "GABUNGAN", gabungan)
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
