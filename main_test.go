package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

var testCtx = context.Background()

func setupTestRedis() *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})
	rdb.FlushDB(testCtx)
	return rdb
}

func TestGenerateRandomString(t *testing.T) {
	length := 8
	randomString := generateRandomString(length)
	assert.Equal(t, length, len(randomString))
}

func TestGenerateUniqueShortURL(t *testing.T) {
	rdb := setupTestRedis()
	defer rdb.Close()

	length := 8
	shortURL := generateUniqueShortURL(testCtx, rdb, length)
	assert.Equal(t, length, len(shortURL))
}

func TestCreateShortURLHandler(t *testing.T) {
	rdb := setupTestRedis()
	defer rdb.Close()

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.POST("/create", func(c *gin.Context) {
		createShortURLHandler(c, rdb)
	})

	w := httptest.NewRecorder()
	body := strings.NewReader("long_url=https://example.com&max_access=10&max_per_hour=5&max_age=3600")
	req, _ := http.NewRequest("POST", "/create", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response["token"])
}

func TestMaxAccess(t *testing.T) {
	rdb := setupTestRedis()
	defer rdb.Close()

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.POST("/create", func(c *gin.Context) {
		createShortURLHandler(c, rdb)
	})

	router.GET("/:token", func(c *gin.Context) {
		redirectHandler(c, rdb)
	})

	w := httptest.NewRecorder()
	body := strings.NewReader("long_url=https://example.com&max_access=10")
	req, _ := http.NewRequest("POST", "/create", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response["token"])

	token := response["token"]
	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/"+token, nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	}
}

func TestMaxPerHour(t *testing.T) {
	rdb := setupTestRedis()
	defer rdb.Close()

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.POST("/create", func(c *gin.Context) {
		createShortURLHandler(c, rdb)
	})

	router.GET("/:token", func(c *gin.Context) {
		redirectHandler(c, rdb)
	})

	w := httptest.NewRecorder()
	body := strings.NewReader("long_url=https://example.com&max_per_hour=5")
	req, _ := http.NewRequest("POST", "/create", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response["token"])

	token := response["token"]
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/"+token, nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	}
}

func TestMaxAge(t *testing.T) {
	rdb := setupTestRedis()
	defer rdb.Close()

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.POST("/create", func(c *gin.Context) {
		createShortURLHandler(c, rdb)
	})

	router.GET("/:token", func(c *gin.Context) {
		redirectHandler(c, rdb)
	})

	w := httptest.NewRecorder()
	body := strings.NewReader("long_url=https://example.com&max_age=1")
	req, _ := http.NewRequest("POST", "/create", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response["token"])

	token := response["token"]
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/"+token, nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)

	// Wait for the token to expire
	<-time.After(2 * time.Second)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/"+token, nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
