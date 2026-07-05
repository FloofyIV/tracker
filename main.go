package main

import (
	"context"
	"log"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	state, err := loadState(cfg.StateFile)
	if err != nil {
		log.Fatalf("failed to load state file %s: %v", cfg.StateFile, err)
	}

	roblox := newRobloxClient(cfg.HTTPTimout, cfg.Verbose)
	discord := newDiscord(cfg.HTTPTimout)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var wg sync.WaitGroup

	if len(cfg.Places) > 0 {
		gt := newGameTracker(cfg, roblox, discord, state)
		wg.Add(1)
		go func() {
			defer wg.Done()
			gt.Run(ctx)
		}()
	}

	if len(cfg.Users) > 0 {
		pt := newPlayerTracker(cfg, roblox, discord, state)
		wg.Add(1)
		go func() {
			defer wg.Done()
			pt.Run(ctx)
		}()
	}

	log.Printf("tracker running, state file is %s", cfg.StateFile)

	<-ctx.Done()
	log.Println("shutdown signal received, stopping.")
	wg.Wait()

	if err := state.save(); err != nil {
		log.Printf("failed to persist final state: %v", err)
	}
	log.Println("stopped cleanly")
}
