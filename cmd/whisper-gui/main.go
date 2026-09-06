package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"whisper-gui/internal/jobmanager"
	"whisper-gui/internal/server"
	"whisper-gui/internal/whisperbin"
)

func main() {
	port := flag.Int("port", 8787, "port to listen on")
	flag.Parse()

	jobs, err := jobmanager.NewManager()
	if err != nil {
		log.Fatalf("failed to start job manager: %v", err)
	}

	mlx := whisperbin.LocateMlxWhisper()
	ffmpeg := whisperbin.LocateFFmpeg()
	whisperCli := whisperbin.LocateWhisperCli()
	log.Printf("mlx_whisper: %s (%s)", describeResolution(mlx), mlx.ResolvedVia)
	log.Printf("ffmpeg:      %s (%s)", describeResolution(ffmpeg), ffmpeg.ResolvedVia)
	log.Printf("whisper-cli: %s (%s)", describeResolution(whisperCli), whisperCli.ResolvedVia)

	addr := fmt.Sprintf(":%d", *port)
	httpServer := server.NewHTTPServer(addr, server.New(jobs).Handler())

	go func() {
		log.Printf("whisper-gui listening on http://localhost:%d", *port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("shutting down...")
	// Force-kill any currently running mlx_whisper process group first --
	// no orphaned subprocess should keep holding GPU unified memory once
	// this server exits.
	jobs.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("http server shutdown error: %v", err)
	}
}

func describeResolution(r whisperbin.Resolution) string {
	if r.ResolvedVia == whisperbin.ViaNone {
		return "未找到"
	}
	return r.Path
}
