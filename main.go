package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	// defaultTerminationDelay delays termination of the program in a graceful shutdown situation.
	// We do this to prevent the pod from exiting immediately upon a pod termination event
	// (e.g. during a rolling update). This gives some time for ingress controllers to react to
	// the Pod IP being removed from the Service's Endpoint list, which prevents traffic from being
	// directed to terminated pods, which otherwise would cause timeout errors and/or request delays.
	// See: https://github.com/kubernetes/ingress-nginx/issues/3335#issuecomment-434970950
	defaultTerminationDelay = 10

	// Choose from one of the following colors:
	// color = "red"
	// color = "orange"
	// color = "yellow"
	// color = "green"
	// color = "blue"
	// this is a test pr2
	color = "purple"
)

func main() {
	var (
		listenAddr       string
		terminationDelay int
	)
	flag.StringVar(&listenAddr, "listen-addr", ":8080", "server listen address")
	flag.IntVar(&terminationDelay, "termination-delay", defaultTerminationDelay, "termination delay in seconds")
	flag.Parse()

	router := http.NewServeMux()
	router.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("./"))))
	router.HandleFunc("/color", getColor)

	server := &http.Server{
		Addr:    listenAddr,
		Handler: router,
	}

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-quit
		server.SetKeepAlivesEnabled(false)
		log.Printf("Signal %v caught. Shutting down in %vs", sig, terminationDelay)
		delay := time.NewTicker(time.Duration(terminationDelay) * time.Second)
		defer delay.Stop()
		select {
		case <-quit:
			log.Println("Second signal caught. Shutting down NOW")
		case <-delay.C:
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		close(done)
	}()

	log.Printf("Started server on %s", listenAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Could not listen on %s: %v\n", listenAddr, err)
	}

	<-done
	log.Println("Server stopped")
}

type colorParameters struct {
	Color                string  `json:"color"`
	DelayLength          float64 `json:"delayLength,omitempty"`
	Return500Probability *int    `json:"return500,omitempty"`
}

func getColor(w http.ResponseWriter, r *http.Request) {
	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
		log.Println(err.Error())
		fmt.Fprintf(w, err.Error())
		return
	}

	var request []colorParameters
	if len(requestBody) > 0 && string(requestBody) != `"[]"` {
		err = json.Unmarshal(requestBody, &request)
		if err != nil {
			w.WriteHeader(500)
			log.Printf("%s: %v", string(requestBody), err.Error())
			fmt.Fprintf(w, err.Error())
			return
		}
	}

	var colorParams colorParameters
	for i := range request {
		cp := request[i]
		if cp.Color == color {
			colorParams = cp
		}
	}

	var delayLength float64
	var delayLengthStr string
	if colorParams.DelayLength > 0 {
		delayLength = colorParams.DelayLength
	}
	if delayLength > 0 {
		delayLengthStr = fmt.Sprintf(" (%fs)", delayLength)
		time.Sleep(time.Duration(delayLength) * time.Second)
	}

	statusCode := http.StatusOK
	if colorParams.Return500Probability != nil && *colorParams.Return500Probability > 0 && *colorParams.Return500Probability >= rand.Intn(100) {
		statusCode = http.StatusInternalServerError
	}
	printColor(color, w, statusCode)
	log.Printf("%d - %s%s\n", statusCode, color, delayLengthStr)
}

func printColor(colorToPrint string, w http.ResponseWriter, statusCode int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(statusCode)
	fmt.Fprintf(w, "\"%s\"", colorToPrint)
}
