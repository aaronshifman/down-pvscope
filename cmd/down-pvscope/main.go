package main

import (
	"context"
	"log"
	"os"

	"github.com/aaronshifman/down-pvscope/pkg/activities"
	"github.com/aaronshifman/down-pvscope/pkg/workflows"
	"github.com/urfave/cli/v3"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "temporal-url",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "temporal-namespace",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			temporal := cmd.String("temporal-url")
			temporalNamespace := cmd.String("temporal-namespace")

			c, err := client.Dial(client.Options{
				HostPort:  temporal,
				Namespace: temporalNamespace,
			})
			if err != nil {
				log.Fatalln("Unable to create Temporal client", err)
			}
			defer c.Close()

			// Create the Temporal worker
			w := worker.New(c, workflows.TaskQueueName, worker.Options{})

			pvcActivities := &activities.PVCActivities{}
			pvActivities := &activities.PVActivities{}
			jobActivities := &activities.JobActivities{}

			// Register Workflow and Activities
			w.RegisterWorkflow(workflows.ScaleDownWorkflow)
			w.RegisterActivity(pvcActivities)
			w.RegisterActivity(pvActivities)
			w.RegisterActivity(jobActivities)

			// Start the Worker
			err = w.Run(worker.InterruptCh())
			if err != nil {
				log.Fatalln("Unable to start Temporal worker", err)
			}

			return nil
		},
	}
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
