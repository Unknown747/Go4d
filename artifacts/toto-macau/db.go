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

// Jadwal 6 sesi Macau 4D per hari (WIB)
var jadwalSesi = []struct {
        sesi int
        jam  string
}{
        {1, "00:01"},
        {2, "13:00"},
        {3, "16:00"},
        {4, "19:00"},
        {5, "22:00"},
        {6, "23:00"},
}

func initDB() {
        var err error
        db, err = sql.Open("sqlite", "./toto.db")
        if err != nil {
                log.Fatal("Failed to open database:", err)
        }
        if err = db.Ping(); err != nil {
                log.Fatal("Failed to connect to database:", err)
        }
        createTables()
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
                        log.Printf("createTables: %v", err)
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
        Total    int     `json:"total"`
        Wins     int     `json:"wins"`
        ShioWins int     `json:"shio_wins"`
        Rate2D   float64 `json:"rate_2d"`
        Rate3D   float64 `json:"rate_3d"`
        Rate4D   float64 `json:"rate_4d"`
        RateShio float64 `json:"rate_shio"`
        Hit2D    int     `json:"hit_2d"`
        Hit3D    int     `json:"hit_3d"`
}

func saveResult(periode int, tanggal string, sesi int, nomor string) error {
        _, err := db.Exec(
                `INSERT INTO results (periode, tanggal, sesi, nomor)
                 VALUES (?, ?, ?, ?)
                 ON CONFLICT(tanggal, sesi) DO UPDATE
                   SET periode = excluded.periode, nomor = excluded.nomor`,
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

func savePredictions(tanggal string, sesi int, metode string, nomorList []string) error {
        _, err := db.Exec(
                `INSERT OR REPLACE INTO predictions (tanggal, sesi, metode, nomor_list)
                 VALUES (?, ?, ?, ?)`,
                tanggal, sesi, metode, strings.Join(nomorList, ","),
        )
        return err
}

func getRecentResults(limit int) []Result {
        rows, err := db.Query(
                `SELECT id, COALESCE(periode,0), tanggal, sesi, nomor, created_at
                 FROM results ORDER BY tanggal DESC, sesi DESC LIMIT ?`,
                limit,
        )
        if err != nil {
                log.Printf("getRecentResults: %v", err)
                return nil
        }
        defer rows.Close()
        var results []Result
        for rows.Next() {
                var r Result
                if err := rows.Scan(&r.ID, &r.Periode, &r.Tanggal, &r.Sesi, &r.Nomor, &r.CreatedAt); err != nil {
                        continue
                }
                results = append(results, r)
        }
        return results
}

func getLatestPredictions(tanggal string, sesi int) []Prediction {
        rows, err := db.Query(
                `SELECT id, tanggal, sesi, metode, nomor_list, created_at
                 FROM predictions WHERE tanggal = ? AND sesi = ? ORDER BY created_at DESC`,
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
                `SELECT id, COALESCE(periode,0), tanggal, sesi, nomor, created_at
                 FROM results WHERE tanggal = ? AND sesi = ?`,
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
                if err := rows.Scan(&tanggal, &sesi, &nomor, &periode, &predList); err != nil {
                        continue
                }

                isHit := false
                hit2d, hit3d := false, false
                if predList != "" {
                        for _, p := range strings.Split(predList, ",") {
                                p = strings.TrimSpace(p)
                                if p == "" {
                                        continue
                                }
                                for len(p) < 4 {
                                        p = "0" + p
                                }
                                act := nomor
                                for len(act) < 4 {
                                        act = "0" + act
                                }
                                if p == act {
                                        isHit = true
                                }
                                if len(p) >= 2 && len(act) >= 2 && p[len(p)-2:] == act[len(act)-2:] {
                                        hit2d = true
                                }
                                if len(p) >= 3 && len(act) >= 3 && p[len(p)-3:] == act[len(act)-3:] {
                                        hit3d = true
                                }
                        }
                }

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
                        "hit_2d":      hit2d,
                        "hit_3d":      hit3d,
                        "shio_hit":    shioHit,
                })
        }
        return history
}

func calculateWinRate() WinRate {
        history := getAllHistory(200)
        wr := WinRate{}
        for _, h := range history {
                pred, _ := h["predictions"].(string)
                if pred == "" {
                        continue
                }
                wr.Total++
                if h["is_hit"].(bool) {
                        wr.Wins++
                }
                if h["hit_2d"].(bool) {
                        wr.Hit2D++
                }
                if h["hit_3d"].(bool) {
                        wr.Hit3D++
                }
                if h["shio_hit"].(bool) {
                        wr.ShioWins++
                }
        }
        if wr.Total > 0 {
                wr.Rate4D = float64(wr.Wins) / float64(wr.Total) * 100
                wr.Rate3D = float64(wr.Hit3D) / float64(wr.Total) * 100
                wr.Rate2D = float64(wr.Hit2D) / float64(wr.Total) * 100
                wr.RateShio = float64(wr.ShioWins) / float64(wr.Total) * 100
        }
        return wr
}

var wib = time.FixedZone("WIB", 7*60*60)

func nowWIB() time.Time {
        return time.Now().In(wib)
}

func todayStr() string {
        return nowWIB().Format("2006-01-02")
}

// nextSessionInfo: cari sesi berikutnya yang belum ada hasilnya
func nextSessionInfo() (string, int) {
        today := todayStr()
        for _, j := range jadwalSesi {
                if _, ok := getTodayResult(today, j.sesi); !ok {
                        return today, j.sesi
                }
        }
        // Semua sesi hari ini sudah selesai → besok sesi 1
        tomorrow := nowWIB().AddDate(0, 0, 1).Format("2006-01-02")
        return tomorrow, 1
}

// getTodayResults: semua hasil hari ini (6 sesi)
func getTodayResults(tanggal string) []Result {
        var res []Result
        for _, j := range jadwalSesi {
                r, ok := getTodayResult(tanggal, j.sesi)
                if ok {
                        res = append(res, r)
                }
        }
        return res
}
