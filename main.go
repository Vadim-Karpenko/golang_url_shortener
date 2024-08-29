package main

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	charset       = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	redisAddr     = "localhost:6379"
	redisPassword = ""
	redisDB       = 0
)

type URL struct {
	Token              string        `json:"token"`
	LongURL            string        `json:"long_url"`
	MaxAccess          int           `json:"max_access"`
	CurrentAccessCount int           `json:"current_access_count"`
	MaxPerHour         int           `json:"max_per_hour"`
	HourlyAccessCount  int           `json:"hourly_access_count"`
	CreatedAt          string        `json:"created_at"`
	LastAccessedAt     string        `json:"last_accessed_at"`
	LastHourlyResetAt  string        `json:"last_hourly_reset_at"`
	AgeDuration        time.Duration `json:"age_duration"`
}

var ctx = context.Background()

// The function generates a random string of a specified length using characters from a given charset.
func generateRandomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// The function generates a unique short URL of a specified length by checking if it already exists in
// a Redis database.
func generateUniqueShortURL(ctx context.Context, rdb *redis.Client, length int) string {
	for {
		shortURL := generateRandomString(length)
		_, err := rdb.Get(ctx, shortURL).Result()
		if err == redis.Nil { // Key doesn't exist
			return shortURL
		}
	}
}

// The `createShortURLHandler` function generates a unique short URL for a given long URL and stores
// the URL entry in Redis with specified parameters.
func createShortURLHandler(c *gin.Context, rdb *redis.Client) {
	longURL := c.PostForm("long_url")
	if longURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Missing long_url parameter"})
		return
	}

	maxAccessInt, err := strconv.Atoi(c.DefaultPostForm("max_access", "-1"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid max_access parameter"})
		return
	}

	maxPerHourInt, err := strconv.Atoi(c.DefaultPostForm("max_per_hour", "-1"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid max_per_hour parameter"})
		return
	}

	maxAgeInt, err := strconv.Atoi(c.DefaultPostForm("max_age", "3600"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid max_age parameter"})
		return
	}

	// Max age can't be less than 1 second and more than 1 year
	if maxAgeInt < 1 || maxAgeInt > 31536000 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid max_age parameter"})
		return
	}

	maxAgeDuration := time.Duration(maxAgeInt) * time.Second
	Token := generateUniqueShortURL(ctx, rdb, 8)

	urlEntry := URL{
		Token:              Token,
		LongURL:            longURL,
		MaxAccess:          maxAccessInt,
		CurrentAccessCount: 0,
		MaxPerHour:         maxPerHourInt,
		CreatedAt:          time.Now().Format(time.RFC3339),
		LastAccessedAt:     time.Now().Format(time.RFC3339),
		LastHourlyResetAt:  time.Now().Format(time.RFC3339),
		AgeDuration:        maxAgeDuration,
	}

	data, err := json.Marshal(urlEntry)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	// Set the key-value pair in Redis
	err = rdb.Set(ctx, Token, data, maxAgeDuration).Err()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": Token})
}

// The `redirectHandler` function retrieves and processes a short URL entry from Redis, updating access
// counts and redirecting to the corresponding long URL. It also checks if the maximum access count or
// maximum access per hour has been reached.
func redirectHandler(c *gin.Context, rdb *redis.Client) {
	token := c.Param("token")

	val, err := rdb.Get(ctx, token).Result()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Error finding your short URL. It may have expired or never existed."})
		return
	}

	var urlEntry URL
	err = json.Unmarshal([]byte(val), &urlEntry)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Error parsing JSON"})
		return
	}

	lastHourlyResetAt, _ := time.Parse(time.RFC3339, urlEntry.LastHourlyResetAt)

	if urlEntry.MaxAccess != -1 && urlEntry.CurrentAccessCount > urlEntry.MaxAccess {
		rdb.Del(ctx, token)
		c.JSON(http.StatusBadRequest, gin.H{"message": "Max access reached"})
		return
	}

	if urlEntry.MaxPerHour != -1 {
		if time.Since(lastHourlyResetAt) >= time.Hour {
			urlEntry.HourlyAccessCount = 0
			urlEntry.LastHourlyResetAt = time.Now().Format(time.RFC3339)
		}

		if urlEntry.HourlyAccessCount >= urlEntry.MaxPerHour {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Max access per hour reached"})
			return
		}
		urlEntry.HourlyAccessCount++
	}

	urlEntry.CurrentAccessCount++
	urlEntry.LastAccessedAt = time.Now().Format(time.RFC3339)

	// Use a goroutine to update Redis asynchronously
	go func() {
		data, _ := json.Marshal(urlEntry)
		rdb.Set(ctx, token, data, urlEntry.AgeDuration)
	}()

	c.Redirect(http.StatusTemporaryRedirect, urlEntry.LongURL)
}

func main() {
	// Uncomment the line below to run the application in release mode
	gin.SetMode(gin.ReleaseMode)
	// The following lines disable logging to stdout and stderr. In case of high traffic, it's recommended
	// to disable logging to prevent the logs from consuming too much resources.
	// gin.DefaultWriter = io.Discard
	// gin.DefaultErrorWriter = io.Discard

	r := gin.Default()

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	r.POST("/create", func(c *gin.Context) {
		createShortURLHandler(c, rdb)
	})

	r.GET("/:token", func(c *gin.Context) {
		redirectHandler(c, rdb)
	})

	r.Run("localhost:8080")
}
