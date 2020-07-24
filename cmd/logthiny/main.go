package main

import (
	"encoding/csv"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/filecoin-project/specs-actors/actors/abi/big"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
)

func main() {
	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	r := csv.NewReader(f)

	// skip the headers
	r.Read()

	minerRewards := make(map[address.Address]big.Int)
	epoch := int64(0)
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}

		if len(record) != 9 {
			log.Fatal("invalid record", record)
		}

		if err != nil {
			log.Fatal(err)
		}
		miner, err := address.NewFromString(record[0])
		if err != nil {
			log.Fatal(err)
		}
		epoch, err = strconv.ParseInt(record[1], 10, 64)
		if err != nil {
			log.Fatal(err)
		}

		rwdPayable, err := types.ParseFIL(record[5])
		if err != nil {
			log.Fatal(err)
		}

		curRwd := types.BigInt(rwdPayable)

		prevRwd, ok := minerRewards[miner]
		if !ok {
			minerRewards[miner] = curRwd
			continue
		}
		minerRewards[miner] = big.Add(prevRwd, curRwd)
	}

	fmt.Println("As of Height ",epoch)
	var ranks []minerRank
	for m, r := range minerRewards {
		ranks = append(ranks, minerRank{
			miner: m,
			rwd:   r,
		})
	}
	sort.Slice(ranks, func(i, j int) bool {
		return ranks[i].rwd.GreaterThan(ranks[j].rwd)
	})

	for rnk, mr := range ranks {
		fmt.Println("Rank ", rnk+1, " Miner ", mr.miner.String(), " Reward ", big.Div(mr.rwd, abi.TokenPrecision).String())
	}
}

type minerRank struct {
	miner address.Address
	rwd big.Int
}
