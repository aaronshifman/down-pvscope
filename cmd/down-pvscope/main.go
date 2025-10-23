package main

import (
	"context"
	"fmt"

	"github.com/aaronshifman/down-pvscope/pkg/activities"
)

func main() {
	ctx := context.Background()
	pv, err := activities.GetMatchingPV(ctx, "bogus", "down-pvscope")
	if err != nil {
		panic(err)
	}
	fmt.Println(pv)

	err = activities.EnsureReclaimPolicyRetain(ctx, pv)
	if err != nil {
		panic(err)
	}
}
