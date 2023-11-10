package main

import (
	"context"
	"fmt"
	"go_http_server_mock_test/request_count"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	intV := getEnv(key, fmt.Sprintf("%d", fallback))
	v, err := strconv.Atoi(intV)
	if err != nil {
		log.Fatal(err)
	}
	return v
}

type netListener struct {
	net.Listener
	rc *request_count.RequestCount
}
type netConn struct {
	net.Conn
	rc *request_count.RequestCount
}

func (nl *netListener) Accept() (net.Conn, error) {
	conn, err := nl.Listener.Accept()
	nl.rc.Increase()
	return &netConn{Conn: conn, rc: nl.rc}, err
}

func (nc *netConn) Close() error {
	nc.rc.Decrease()
	return nc.Conn.Close()
}

type Query struct {
	Delay uint8 `form:"delay"`
}

func main() {
	router := gin.Default()
	router.GET("/", func(c *gin.Context) {
		var query Query
		err := c.ShouldBindQuery(&query)
		if err != nil {
			log.Println(err)
		}

		requestAt := time.Now()

		time.Sleep(time.Duration(query.Delay) * time.Second)

		c.JSON(http.StatusOK, struct {
			RequestAt  time.Time
			ResponseAt time.Time
		}{
			RequestAt:  requestAt,
			ResponseAt: time.Now(),
		})
	})

	srv := &http.Server{
		Handler: router,
	}

	requestC := request_count.New()

	go func() {
		// service connections
		// Wrap the original listener with our counting listener
		// Set running port via PORT env, default 8080
		portStr := getEnv("PORT", "8080")
		listener, err := net.Listen("tcp", fmt.Sprintf(":%s", portStr))
		if err != nil {
			log.Fatalf("Error when creating listener: %s\n", err)
		}
		countingListener := &netListener{
			Listener: listener,
			rc:       requestC,
		}

		log.Printf("Server is running on the TCP port: %s", portStr)

		if err := srv.Serve(countingListener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen error: %s\n", err)
		}
	}()
	// Wait for interrupt signal to gracefully shutdown the server with
	quit := make(chan os.Signal)
	// kill (no param) default send syscanll.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall. SIGKILL but can"t be catch, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	log.Printf("Shutting down the server due to the %v signal", <-quit)
	log.Println("Server is going to be shutdown as soon as possible when all handled requests are finished")
	log.Printf("Current requests on serving: %d", requestC.Count())
	gracefullyShutdownTimeout := getEnvInt("GRACEFULLY_SHUTDOWN_TIMEOUT", 0)
	switch gracefullyShutdownTimeout {
	case 0:
		// In kubernetes, they don't respect the timeout of our app, they just notify and SIGKILL after reaching `terminationGracePeriodSeconds` to all containers in the pod
		// That's why the server must be shutdown ASAP
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Fatal("Server shutdown error:", err)
		}
	default:
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(gracefullyShutdownTimeout)*time.Second)
		defer cancel()

		log.Printf("Server is going to be shutdown after %d seconds", gracefullyShutdownTimeout)

		// After reaching the timeout the server is going to be shutdown, all handled requests will be dropped, response will be "(52) Empty reply from server"
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatal("Server shutdown: ", err)
		}
	}

	log.Println("Server exiting")
}
