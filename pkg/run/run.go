// Package run manages Read, Write and Signal goroutines.
package run

import (
	"context"
	"fmt"
	"os"

	"github.com/mediocregopher/radix/v3"
	"golang.org/x/sync/errgroup"

	"github.com/stickermule/rump/pkg/config"
	"github.com/stickermule/rump/pkg/file"
	"github.com/stickermule/rump/pkg/message"
	"github.com/stickermule/rump/pkg/redis"
	"github.com/stickermule/rump/pkg/signal"
)

// Connect redis
func connect(rsc config.Resource) (*radix.Pool, error) {
	uri, opts := rsc.URI, []radix.DialOpt{}
	if rsc.DB > 0 {
		opts = append(opts, radix.DialSelectDB(rsc.DB))
	}
	if rsc.Password != "" {
		if rsc.Username != "" {
			opts = append(opts, radix.DialAuthUser(rsc.Username, rsc.Password))
		} else {
			opts = append(opts, radix.DialAuthPass(rsc.Password))
		}
	}
	poolOpts := []radix.PoolOpt{}
	if len(opts) > 0 {
		poolOpts = append(poolOpts, radix.PoolConnFunc(func(network, addr string) (radix.Conn, error) {
			return radix.Dial(network, addr, opts...)
		}))
	}
	return radix.NewPool("tcp", uri, 1, poolOpts...)
}

// Exit helper
func exit(e error) {
	fmt.Println(e)
	os.Exit(1)
}

// Run orchestrate the Reader, Writer and Signal handler.
func Run(cfg config.Config) {
	// create ErrGroup to manage goroutines
	ctx, cancel := context.WithCancel(context.Background())
	g, gctx := errgroup.WithContext(ctx)

	// Start signal handling goroutine
	g.Go(func() error {
		return signal.Run(gctx, cancel)
	})

	// Create shared message bus
	ch := make(message.Bus, 100)

	// Create and run either a Redis or File Source reader.
	if cfg.Source.IsRedis {
		db, err := connect(cfg.Source)
		if err != nil {
			exit(err)
		}

		source := redis.New(db, ch, cfg.Match, !cfg.Verbose, cfg.TTL)

		g.Go(func() error {
			return source.Read(gctx)
		})
	} else {
		source := file.New(cfg.Source.URI, ch, !cfg.Verbose, cfg.TTL)

		g.Go(func() error {
			return source.Read(gctx)
		})
	}

	// Create and run either a Redis or File Target writer.
	if cfg.Target.IsRedis {
		db, err := connect(cfg.Target)
		if err != nil {
			exit(err)
		}

		target := redis.New(db, ch, cfg.Match, !cfg.Verbose, cfg.TTL)

		g.Go(func() error {
			defer cancel()
			return target.Write(gctx)
		})
	} else {
		target := file.New(cfg.Target.URI, ch, !cfg.Verbose, cfg.TTL)

		g.Go(func() error {
			defer cancel()
			return target.Write(gctx)
		})
	}

	// Block and wait for goroutines
	err := g.Wait()
	if err != nil && err != context.Canceled {
		exit(err)
	} else {
		fmt.Println("done")
	}
}
