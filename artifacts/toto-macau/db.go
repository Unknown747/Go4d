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
                `CREATE INDEX IF NOT EXISTS idx_results_tanggal ON results(tanggal)`,
        }
        for _, q := range queries {
                if _, err := db.Exec(q); err != nil {
                        log.Printf("Note creating table/index: %v", err)
                }
        }
}

// migrateDB adds columns that may not exist in older schema.
// SQLite has no ALTER TABLE … ADD COLUMN IF NOT EXISTS, so we check PRAGMA first.
func migrateDB() {
        rows, err := db.Query(`PRAGMA table_info(results)`)
        if err != nil {
                log.Printf("migrateDB: cannot read table_info: %v", err)
                return
        }
        hasPeriode := false
        for rows.Next() {
                var cid, notNull, pk int
                var name, colType string
                var dflt interface{}
                if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err == nil && name == "periode" {
                        hasPeriode = true
                }
        }
        rows.Close()
        if !hasPeriode {
                if _, err := db.Exec(`ALTER TABLE results ADD COLUMN periode INTEGER`); err != nil {
                        log.Printf("migrateDB: failed to add periode column: %v", err)
                }
        }
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
        _, err := db.Exec(
                `INSERT OR IGNORE INTO predictions (tanggal, sesi, metode, nomor_list) VALUES (?, ?, ?, ?)`,
                tanggal, sesi, metode, strings.Join(nomorList, ","),
        )
        return err
}

func getRecentResults(limit int) []Result {
        rows, err := db.Query(
                `SELECT id, COALESCE(periode,0), tanggal, sesi, nomor, created_at FROM results ORDER BY tanggal DESC, sesi DESC LIMIT ?`,
                limit,
        )
        if err != nil {
                log.Printf("getRecentResults: query error: %v", err)
                return nil
        }
        defer rows.Close()

        var results []Result
        for rows.Next() {
                var r Result
                if err := rows.Scan(&r.ID, &r.Periode, &r.Tanggal, &r.Sesi, &r.Nomor, &r.CreatedAt); err != nil {
                        log.Printf("getRecentResults: scan error: %v", err)
                        continue
                }
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
                if err := rows.Scan(&p.ID, &p.Tanggal, &p.Sesi, &p.Metode, &p.NomorList, &p.CreatedAt); err != nil {
                        log.Printf("getLatestPredictions: scan error: %v", err)
                        continue
                }
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

// WIB = UTC+7
var wib = time.FixedZone("WIB", 7*60*60)

func nowWIB() time.Time {
        return time.Now().In(wib)
}

func todayStr() string {
        return nowWIB().Format("2006-01-02")
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

        tomorrow := nowWIB().AddDate(0, 0, 1).Format("2006-01-02")
        return tomorrow, 1
}

// DayPair menyimpan pasangan hasil sesi 1 & sesi 2 dalam satu hari
type DayPair struct {
        Tanggal string
        Sesi1   string
        Sesi2   string
}

// getDayPairs mengambil pasangan (sesi1, sesi2) dari hari yang sama
func getDayPairs(limit int) []DayPair {
        rows, err := db.Query(`
                SELECT r1.tanggal, r1.nomor, r2.nomor
                FROM results r1
                JOIN results r2 ON r1.tanggal = r2.tanggal AND r1.sesi = 1 AND r2.sesi = 2
                ORDER BY r1.tanggal DESC
                LIMIT ?
        `, limit)
        if err != nil {
                return nil
        }
        defer rows.Close()

        var pairs []DayPair
        for rows.Next() {
                var p DayPair
                rows.Scan(&p.Tanggal, &p.Sesi1, &p.Sesi2)
                pairs = append(pairs, p)
        }
        return pairs
}

