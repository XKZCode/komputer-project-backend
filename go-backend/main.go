package main

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

type Incident struct {
	ID               int       `json:"id"`
	AudioPath        string    `json:"audio_path"`
	TranscriptText   string    `json:"transcript_text"`
	ThreatLevel      string    `json:"threat_level"`
	KeywordsDetected string    `json:"keywords_detected"`
	Latitude         float64   `json:"latitude"`
	Longitude        float64   `json:"longitude"`
	ReviewStatus     string    `json:"review_status"`
	CreatedAt        time.Time `json:"created_at"`
}

type STTResponse struct {
	Status     string `json:"status"`
	Transcript string `json:"transcript"`
}

var db *sql.DB

func main() {
	var err error
	// Koneksi ke MySQL
	dsn := "root:@tcp(db:3306)/ht_monitoring_db?parseTime=true"
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Gagal koneksi database:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Database tidak merespon:", err)
	}
	fmt.Println("Berhasil terhubung ke MySQL Database!")

	router := gin.Default()

	// Middleware CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.StaticFile("/tester", "tester.html")

	// Endpoints API
	router.POST("/api/audio/upload", uploadAudioHandler)
	router.GET("/api/incidents", getIncidentsHandler) // Sudah mendukung Filter Query
	router.PATCH("/api/incidents/:id/status", updateStatusHandler)
	router.GET("/api/incidents/export", exportCSVHandler) // FITUR BARU: Ekspor CSV

	fmt.Println("Server Go Backend berjalan di port 8080...")
	router.Run(":8080")
}

// Pengayaan Kamus Kata Kunci Ancaman & Kode Sandi HT
func analyzeThreat(transcript string) (string, string) {
	lower := strings.ToLower(transcript)

	// Sandi HT & kata bahaya tingkat tinggi / ekstrem
	ekstremKeywords := []string{"kebakaran", "pembegalan", "penculikan", "pembunuhan", "teror", "bom", "8-1"}
	tinggiKeywords := []string{"pencurian", "maling", "perampokan", "kecelakaan", "tawuran", "senjata", "darurat", "8-6"}

	var detected []string
	isEkstrem := false
	isTinggi := false

	for _, kw := range ekstremKeywords {
		if strings.Contains(lower, kw) {
			detected = append(detected, kw)
			isEkstrem = true
		}
	}

	for _, kw := range tinggiKeywords {
		if strings.Contains(lower, kw) {
			detected = append(detected, kw)
			isTinggi = true
		}
	}

	if len(detected) == 0 {
		return "Aman", "-"
	}

	keywordsStr := strings.Join(detected, ", ")
	if isEkstrem {
		return "Ekstrem", keywordsStr
	}
	if isTinggi {
		return "Tinggi", keywordsStr
	}

	return "Sedang", keywordsStr
}

// Auto-Delete File Audio (Retensi Maksimal 10 File)
func cleanupOldAudioFiles(uploadDir string, maxFiles int) {
	files, err := os.ReadDir(uploadDir)
	if err != nil {
		return
	}

	var audioFiles []os.DirEntry
	for _, f := range files {
		if !f.IsDir() {
			audioFiles = append(audioFiles, f)
		}
	}

	if len(audioFiles) >= maxFiles {
		sort.Slice(audioFiles, func(i, j int) bool {
			infoI, _ := audioFiles[i].Info()
			infoJ, _ := audioFiles[j].Info()
			return infoI.ModTime().Before(infoJ.ModTime())
		})

		numToDelete := len(audioFiles) - maxFiles + 1
		for i := 0; i < numToDelete; i++ {
			filePath := filepath.Join(uploadDir, audioFiles[i].Name())
			if err := os.Remove(filePath); err == nil {
				fmt.Printf("[Auto-Cleanup] Berhasil menghapus audio lama: %s\n", filePath)
			}
		}
	}
}

func uploadAudioHandler(c *gin.Context) {
	os.MkdirAll("uploads", os.ModePerm)
	cleanupOldAudioFiles("uploads", 10)

	file, err := c.FormFile("audio")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File audio wajib diunggah"})
		return
	}

	lat := c.DefaultPostForm("latitude", "0.0")
	long := c.DefaultPostForm("longitude", "0.0")

	filename := fmt.Sprintf("%d_%s", time.Now().Unix(), file.Filename)
	savePath := filepath.Join("uploads", filename)
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan audio"})
		return
	}

	sttTranscript, err := sendToPythonSTT(savePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses STT: " + err.Error()})
		return
	}

	threatLevel, keywords := analyzeThreat(sttTranscript)

	query := `INSERT INTO audio_transcripts (audio_path, transcript_text, threat_level, keywords_detected, latitude, longitude) VALUES (?, ?, ?, ?, ?, ?)`
	res, err := db.Exec(query, savePath, sttTranscript, threatLevel, keywords, lat, long)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan ke DB: " + err.Error()})
		return
	}

	id, _ := res.LastInsertId()

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil diproses",
		"data": gin.H{
			"id":            id,
			"transcript":    sttTranscript,
			"threat_level":  threatLevel,
			"keywords":      keywords,
			"trigger_siren": threatLevel == "Tinggi" || threatLevel == "Ekstrem",
		},
	})
}

func sendToPythonSTT(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return "", err
	}
	writer.Close()

	req, err := http.NewRequest("POST", "http://python-stt:8000/process-audio", body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var sttRes STTResponse
	if err := json.NewDecoder(resp.Body).Decode(&sttRes); err != nil {
		return "", err
	}

	return sttRes.Transcript, nil
}

func getIncidentsHandler(c *gin.Context) {
	threatLevel := c.Query("threat_level")
	reviewStatus := c.Query("review_status")

	query := "SELECT id, audio_path, transcript_text, threat_level, keywords_detected, latitude, longitude, review_status, created_at FROM audio_transcripts WHERE 1=1"
	var args []interface{}

	if threatLevel != "" {
		query += " AND threat_level = ?"
		args = append(args, threatLevel)
	}
	if reviewStatus != "" {
		query += " AND review_status = ?"
		args = append(args, reviewStatus)
	}

	query += " ORDER BY created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var incidents []Incident
	for rows.Next() {
		var inc Incident
		rows.Scan(&inc.ID, &inc.AudioPath, &inc.TranscriptText, &inc.ThreatLevel, &inc.KeywordsDetected, &inc.Latitude, &inc.Longitude, &inc.ReviewStatus, &inc.CreatedAt)
		incidents = append(incidents, inc)
	}

	c.JSON(http.StatusOK, gin.H{"data": incidents})
}

func updateStatusHandler(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		ReviewStatus string `json:"review_status"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format JSON salah"})
		return
	}

	_, err := db.Exec("UPDATE audio_transcripts SET review_status = ? WHERE id = ?", body.ReviewStatus, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status berhasil diperbarui"})
}

// Handler Baru: Ekspor Data ke File CSV
func exportCSVHandler(c *gin.Context) {
	c.Header("Content-Disposition", "attachment; filename=laporan_insiden_ht.csv")
	c.Header("Content-Type", "text/csv")

	rows, err := db.Query("SELECT id, created_at, transcript_text, threat_level, keywords_detected, latitude, longitude, review_status FROM audio_transcripts ORDER BY created_at DESC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data untuk ekspor"})
		return
	}
	defer rows.Close()

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Header CSV
	writer.Write([]string{"ID", "Waktu", "Transkrip Suara", "Tingkat Bahaya", "Kata Kunci", "Latitude", "Longitude", "Status Review"})

	for rows.Next() {
		var id int
		var createdAt, transcriptText, threatLvl, keywords, reviewStat string
		var lat, long float64

		rows.Scan(&id, &createdAt, &transcriptText, &threatLvl, &keywords, &lat, &long, &reviewStat)

		writer.Write([]string{
			strconv.Itoa(id),
			createdAt,
			transcriptText,
			threatLvl,
			keywords,
			fmt.Sprintf("%f", lat),
			fmt.Sprintf("%f", long),
			reviewStat,
		})
	}
}
