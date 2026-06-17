package main

import (
        "database/sql"
        "fmt"
        "log"
        "strings"
        "time"

        _ "modernc.org/sqlite"
)

var db *sql.DB

func initDB() {
        var err error
        db, err = sql.Open("sqlite", "./toto.db")
        if err != nil {
                log.Fatal("Failed to open database:", err)
        }

        createTables()
        migrateDB()
}

func createTables() {
        queries := []string{
                `CREATE TABLE IF NOT EXISTS results (
                        id INTEGER PRIMARY KEY AUTOINCREMENT,
                        periode INTEGER,
                        tanggal TEXT NOT NULL,
                        sesi INTEGER NOT NULL,
                        nomor TEXT NOT NULL,
                        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                        UNIQUE(tanggal, sesi)
                )`,
                `CREATE TABLE IF NOT EXISTS predictions (
                        id INTEGER PRIMARY KEY AUTOINCREMENT,
                        tanggal TEXT NOT NULL,
                        sesi INTEGER NOT NULL,
                        metode TEXT NOT NULL,
                        nomor_list TEXT NOT NULL,
                        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
                )`,
                `CREATE UNIQUE INDEX IF NOT EXISTS idx_pred_once ON predictions(tanggal, sesi, metode)`,
        }
        for _, q := range queries {
                if _, err := db.Exec(q); err != nil {
                        log.Printf("Note creating table/index: %v", err)
                }
        }
}

// migrateDB adds columns that may not exist in older schema
func migrateDB() {
        db.Exec(`ALTER TABLE results ADD COLUMN periode INTEGER`)
}

type Result struct {
        ID        int    `json:"id"`
        Periode   int    `json:"periode"`
        Tanggal   string `json:"tanggal"`
        Sesi      int    `json:"sesi"`
        Nomor     string `json:"nomor"`
        CreatedAt string `json:"created_at"`
}

type Prediction struct {
        ID        int    `json:"id"`
        Tanggal   string `json:"tanggal"`
        Sesi      int    `json:"sesi"`
        Metode    string `json:"metode"`
        NomorList string `json:"nomor_list"`
        CreatedAt string `json:"created_at"`
}

type WinRate struct {
        Total        int     `json:"total"`
        Wins         int     `json:"wins"`
        ShioWins     int     `json:"shio_wins"`
        RateExact    float64 `json:"rate_exact"`
        RateShio     float64 `json:"rate_shio"`
}

func saveResult(periode int, tanggal string, sesi int, nomor string) error {
        _, err := db.Exec(
                `INSERT OR REPLACE INTO results (periode, tanggal, sesi, nomor) VALUES (?, ?, ?, ?)`,
                periode, tanggal, sesi, nomor,
        )
        return err
}

func updatePeriode(tanggal string, sesi int, periode int) error {
        _, err := db.Exec(
                `UPDATE results SET periode = ? WHERE tanggal = ? AND sesi = ?`,
                periode, tanggal, sesi,
        )
        return err
}

// savePredictions uses INSERT OR IGNORE so predictions are written only once per session/method
func savePredictions(tanggal string, sesi int, metode string, nomorList []string) error {
        joined := joinStrings(nomorList, ",")
        _, err := db.Exec(
                `INSERT OR IGNORE INTO predictions (tanggal, sesi, metode, nomor_list) VALUES (?, ?, ?, ?)`,
                tanggal, sesi, metode, joined,
        )
        return err
}

func getRecentResults(limit int) []Result {
        rows, err := db.Query(
                `SELECT id, COALESCE(periode,0), tanggal, sesi, nomor, created_at FROM results ORDER BY tanggal DESC, sesi DESC LIMIT ?`,
                limit,
        )
        if err != nil {
                return nil
        }
        defer rows.Close()

        var results []Result
        for rows.Next() {
                var r Result
                rows.Scan(&r.ID, &r.Periode, &r.Tanggal, &r.Sesi, &r.Nomor, &r.CreatedAt)
                results = append(results, r)
        }
        return results
}

func getLatestPredictions(tanggal string, sesi int) []Prediction {
        rows, err := db.Query(
                `SELECT id, tanggal, sesi, metode, nomor_list, created_at FROM predictions WHERE tanggal = ? AND sesi = ? ORDER BY created_at DESC`,
                tanggal, sesi,
        )
        if err != nil {
                return nil
        }
        defer rows.Close()

        var preds []Prediction
        seen := map[string]bool{}
        for rows.Next() {
                var p Prediction
                rows.Scan(&p.ID, &p.Tanggal, &p.Sesi, &p.Metode, &p.NomorList, &p.CreatedAt)
                key := fmt.Sprintf("%s_%d_%s", p.Tanggal, p.Sesi, p.Metode)
                if !seen[key] {
                        seen[key] = true
                        preds = append(preds, p)
                }
        }
        return preds
}

func getTodayResult(tanggal string, sesi int) (Result, bool) {
        var r Result
        err := db.QueryRow(
                `SELECT id, COALESCE(periode,0), tanggal, sesi, nomor, created_at FROM results WHERE tanggal = ? AND sesi = ?`,
                tanggal, sesi,
        ).Scan(&r.ID, &r.Periode, &r.Tanggal, &r.Sesi, &r.Nomor, &r.CreatedAt)
        if err != nil {
                return r, false
        }
        return r, true
}

func getAllHistory(limit int) []map[string]interface{} {
        rows, err := db.Query(`
                SELECT r.tanggal, r.sesi, r.nomor, COALESCE(r.periode,0),
                       COALESCE(p.nomor_list, '') as pred_list
                FROM results r
                LEFT JOIN predictions p ON r.tanggal = p.tanggal AND r.sesi = p.sesi AND p.metode = 'GABUNGAN'
                ORDER BY r.tanggal DESC, r.sesi DESC
                LIMIT ?
        `, limit)
        if err != nil {
                return nil
        }
        defer rows.Close()

        var history []map[string]interface{}
        for rows.Next() {
                var tanggal, nomor, predList string
                var sesi, periode int
                rows.Scan(&tanggal, &sesi, &nomor, &periode, &predList)

                // Check if result is an exact hit in predictions
                isHit := false
                if predList != "" {
                        for _, p := range strings.Split(predList, ",") {
                                if strings.TrimSpace(p) == nomor {
                                        isHit = true
                                        break
                                }
                        }
                }

                // Check shio hit
                shioHit := false
                predShio := shioOf(nomor)
                if predList != "" {
                        for _, p := range strings.Split(predList, ",") {
                                p = strings.TrimSpace(p)
                                if p != "" && shioOf(p) == predShio {
                                        shioHit = true
                                        break
                                }
                        }
                }

                history = append(history, map[string]interface{}{
                        "tanggal":     tanggal,
                        "sesi":        sesi,
                        "nomor":       nomor,
                        "periode":     periode,
                        "predictions": predList,
                        "is_hit":      isHit,
                        "shio_hit":    shioHit,
                })
        }
        return history
}

func calculateWinRate() WinRate {
        history := getAllHistory(100)

        wr := WinRate{}
        for _, h := range history {
                pred, _ := h["predictions"].(string)
                if pred == "" {
                        continue
                }
                nomor, _ := h["nomor"].(string)
                wr.Total++

                // Exact hit
                for _, p := range strings.Split(pred, ",") {
                        if strings.TrimSpace(p) == nomor {
                                wr.Wins++
                                break
                        }
                }

                // Shio hit
                resultShio := shioOf(nomor)
                for _, p := range strings.Split(pred, ",") {
                        p = strings.TrimSpace(p)
                        if p != "" && shioOf(p) == resultShio {
                                wr.ShioWins++
                                break
                        }
                }
        }

        if wr.Total > 0 {
                wr.RateExact = float64(wr.Wins) / float64(wr.Total) * 100
                wr.RateShio = float64(wr.ShioWins) / float64(wr.Total) * 100
        }
        return wr
}

func todayStr() string {
        return time.Now().Format("2006-01-02")
}

func nextSessionInfo() (string, int) {
        today := todayStr()

        _, s1ok := getTodayResult(today, 1)
        if !s1ok {
                return today, 1
        }
        _, s2ok := getTodayResult(today, 2)
        if !s2ok {
                return today, 2
        }

        tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
        return tomorrow, 1
}

func joinStrings(ss []string, sep string) string {
        result := ""
        for i, s := range ss {
                if i > 0 {
                        result += sep
                }
                result += s
        }
        return result
}

func splitString(s, sep string) []string {
        if s == "" {
                return nil
        }
        result := []string{}
        start := 0
        for i := 0; i <= len(s)-len(sep); i++ {
                if s[i:i+len(sep)] == sep {
                        result = append(result, s[start:i])
                        start = i + len(sep)
                        i += len(sep) - 1
                }
        }
        result = append(result, s[start:])
        return result
}
