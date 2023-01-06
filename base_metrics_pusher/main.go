package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"log"
	"strconv"
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
	ddb_client := dynamodb.NewFromConfig(cfg)

	namespace := aws.String("Blog")

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
		Namespace:  namespace,
		MetricData: datums,
	})
	if err != nil {
		log.Fatal(err)
		return
	} else {
		fmt.Printf("Sent %d events to CW\n", len(files))
	}

	clients := get_all_clients()

	datums = make([]types.MetricDatum, len(clients))

	for i, client := range clients {
		datums[i] = types.MetricDatum{
			MetricName: aws.String("requests"),
			Timestamp:  &t,
			Value:      aws.Float64(0),
			Unit:       types.StandardUnitCount,
			Dimensions: []types.Dimension{
				{
					Name:  aws.String("client"),
					Value: aws.String(client),
				},
			},
		}
	}

	_, err = cw_client.PutMetricData(context.TODO(), &cloudwatch.PutMetricDataInput{
		Namespace:  namespace,
		MetricData: datums,
	})
	if err != nil {
		log.Fatal(err)
		return
	} else {
		fmt.Printf("Sent %d events to CW\n", len(clients))
	}

	scan_result, err := ddb_client.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName: aws.String("BlogRSSSubscriptions"),
	})
	if err != nil {
		log.Fatal(err)
		return
	}
	subscribers := 0
	for _, item := range scan_result.Items {
		i, err := strconv.Atoi(item["Count"].(*ddbtypes.AttributeValueMemberN).Value)
		if err != nil {
			log.Fatal(err)
			return
		}
		subscribers += i
	}
	cw_client.PutMetricData(context.TODO(), &cloudwatch.PutMetricDataInput{
		Namespace: namespace,
		MetricData: []types.MetricDatum{
			{
				MetricName: aws.String("feed-subscribers"),
				Timestamp:  &t,
				Value:      aws.Float64(float64(subscribers)),
				Unit:       types.StandardUnitCount,
			},
		},
	})
	fmt.Printf("Pushed subscriber metric: %d\n", subscribers)
}

func main() {
	// Make the handler available for Remote Procedure Call by AWS Lambda
	lambda.Start(handler)
}
