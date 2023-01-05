package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"log"
	"time"
)

type MyEvent struct{}

func handler(ctx context.Context) {
	files := get_all_files()

	// Load the Shared AWS Configuration (~/.aws/config)
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	cw_client := cloudwatch.NewFromConfig(cfg)

	var namespace string = "Blog"

	datums := make([]types.MetricDatum, len(files))

	t := time.Now()

	for i, file := range files {
		datums[i] = types.MetricDatum{
			MetricName: aws.String("requests"),
			Timestamp:  &t,
			Value:      aws.Float64(0),
			Unit:       types.StandardUnitCount,
			Dimensions: []types.Dimension{
				{
					Name:  aws.String("path"),
					Value: aws.String(file),
				},
			},
		}
	}

	_, err = cw_client.PutMetricData(context.TODO(), &cloudwatch.PutMetricDataInput{
		Namespace:  &namespace,
		MetricData: datums,
	})
	if err != nil {
		log.Fatal(err)
		return
	} else {
		fmt.Printf("Sent %d events to CW\n", len(files))
	}
}

func main() {
	// Make the handler available for Remote Procedure Call by AWS Lambda
	lambda.Start(handler)
}
