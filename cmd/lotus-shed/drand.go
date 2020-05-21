package main

import (
	"context"
	"sync"
	"time"

	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/beacon/drand"

	"gopkg.in/urfave/cli.v2"
)

var drandCmd = &cli.Command{
	Name:        "drand",
	Description: "create drand beacons and request entries every blocktime",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "beacons",
			Value: 16,
		},
		&cli.DurationFlag{
			Name:  "delay",
			Value: time.Millisecond * 16,
		},
		&cli.DurationFlag{
			Name:  "interval",
			Value: build.BlockDelay * time.Second,
		},
	},
	Action: func(cctx *cli.Context) error {
		var wg sync.WaitGroup
		for i := 0; i < cctx.Int("beacons"); i++ {
			time.Sleep(cctx.Duration("delay"))
			log.Infof("Starting %d", i)
			wg.Add(1)
			go func() {
				defer wg.Done()

				db, err := drand.NewDrandBeacon(1588122000, build.BlockDelay)
				if err != nil {
					log.Errorf("error creating beacon %s", err)
				}

				ticker := time.NewTicker(cctx.Duration("interval"))

				r := uint64(0)

				for {
					select {
					case _ = <-ticker.C:
						r = r + 1
						c := db.Entry(context.Background(), r)

						brsp := <-c

						if brsp.Err != nil {
							log.Errorf("error on entry response %s", err)
							continue
						}

						break
					}
				}
			}()
		}

		wg.Wait()

		return nil
	},
}
