package main

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

type metrics_batch struct {
	metrics     []metrics_record
	subscribers map[string]int
}

type metrics_record struct {
	Page      string
	Time      time.Time
	UserAgent string
}

func is_rss(user_agent string) bool {
	return strings.HasPrefix(user_agent, "Feedbin") || strings.HasPrefix(user_agent, "Feedly/")
}

func get_feed_subscribers(user_agent string) map[string]int {
	ret := make(map[string]int)

	if !is_rss(user_agent) {
		return ret
	}

	if strings.HasPrefix(user_agent, "Feedbin") {
		num, err := strconv.Atoi(strings.Split(user_agent, "%20")[3])
		if err == nil {
			ret["Feedbin"] = num
		}
	}

	if strings.HasPrefix(user_agent, "Feedly") {
		num, err := strconv.Atoi(strings.Split(user_agent, "%20")[2])
		if err == nil {
			ret["Feedly"] = num
		}
	}

	return ret
}

func normalize_user_agent(user_agent string) (string, error) {
	if strings.Contains(user_agent, "Mastodon/") && strings.HasSuffix(user_agent, "Bot") {
		return "Mastodon Bot", nil // UA
	}

	if strings.HasPrefix(user_agent, "Twitterbot/") {
		return "Twitter Bot", nil // UA
	}

	if strings.Contains(user_agent, "iPhone;") {
		return "iPhone", nil // UA
	}

	if strings.Contains(user_agent, "Macintosh;") {
		return "Mac", nil // UA
	}

	if strings.HasPrefix(user_agent, "Pleroma") {
		return "Mastodon Bot", nil // UA
	}

	if is_rss(user_agent) {
		return "RSS Feed Reader", nil // UA
	}

	if strings.HasPrefix(user_agent, "Akkoma") {
		return "Mastodon Bot", nil // UA
	}

	if strings.HasPrefix(user_agent, "Expanse") ||
		strings.HasSuffix(user_agent, "ahrefs.com/robot/)") ||
		strings.Contains(user_agent, "YandexBot") ||
		strings.Contains(user_agent, "Googlebot") ||
		strings.Contains(user_agent, "TrendsmapResolver") {
		return "Scanners/Crawlers", nil // UA
	}

	if strings.Contains(user_agent, "%20Linux%20") {
		return "Linux", nil // UA
	}

	return "", errors.New(fmt.Sprintf("Unknown UA: %s", user_agent))
}

func handle_record(s3_client *s3.Client, key *string, batches chan metrics_batch) {
	ret := make(map[string]int)

	object, err := s3_client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String("run-parallel.sh-logs"),
		Key:    key,
	})

	if err != nil {
		log.Fatal(err)
		batches <- metrics_batch{}
		return
	}

	reader, err := gzip.NewReader(object.Body)
	if err != nil {
		log.Fatal(err)
		batches <- metrics_batch{}
		return
	}

	bytes, err := io.ReadAll(reader)
	if err != nil {
		log.Fatal(err)
		batches <- metrics_batch{}
		return
	}

	log_data := string(bytes)
	log.Printf("log_data (uncompressed) is %d bytes", len(log_data))
	log.Print(log_data)

	lines := strings.Split(log_data, "\n")
	records := make([]metrics_record, 0, len(lines))

	for _, line := range lines {
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		var datetime_parts []string = parts[0:2]
		uri := parts[7]
		user_agent := parts[10]

		for k, v := range get_feed_subscribers(user_agent) {
			ret[k] = v
		}

		time_string := fmt.Sprintf("%s:%s UTC", datetime_parts[0], datetime_parts[1])
		log.Printf("Request for %s at %s from %s", uri, time_string, user_agent)
		t, err := time.Parse("2006-01-02:15:04:05 MST", time_string)
		if err != nil {
			log.Fatal(err)
			batches <- metrics_batch{}
			return
		}
		log.Printf("Time as parsed is %s", t.Format(time.UnixDate))
		var metric_url = strings.TrimSuffix(fmt.Sprintf("path:%s", uri), "/")
		log.Printf("Metric URL is %s", metric_url)
		records = append(records, metrics_record{Page: metric_url, Time: t, UserAgent: user_agent})
	}
	batches <- metrics_batch{metrics: records, subscribers: ret}
}

func push_subscribers_to_ddb(ddb_client *dynamodb.Client, subscribers map[string]int, done chan bool) {
	defer close(done)

	if len(subscribers) == 0 {
		return
	}

	ddb_writes := make([]ddbtypes.WriteRequest, 0, len(subscribers))
	for aggregator, count := range subscribers {
		attrs := make(map[string]ddbtypes.AttributeValue)
		attrs["Count"] = &ddbtypes.AttributeValueMemberN{Value: strconv.Itoa(count)}
		attrs["Aggregator"] = &ddbtypes.AttributeValueMemberS{Value: aggregator}

		ddb_writes = append(ddb_writes, ddbtypes.WriteRequest{PutRequest: &ddbtypes.PutRequest{Item: attrs}})
	}

	batch_ddb_writes := make(map[string][]ddbtypes.WriteRequest)
	batch_ddb_writes["BlogRSSSubscriptions"] = ddb_writes
	_, err := ddb_client.BatchWriteItem(context.TODO(), &dynamodb.BatchWriteItemInput{RequestItems: batch_ddb_writes})
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Printf("Pushed subscribers to DDB")
	}
}

func push_metrics_to_cloudwatch(wg *sync.WaitGroup, cw_client *cloudwatch.Client, events []metrics_record) {
	defer wg.Done()

	if len(events) == 0 {
		return
	}

	fmt.Printf("Sending %d events to CW\n", len(events))

	datums := make([]types.MetricDatum, len(events)*2)

	for i, event := range events {
		ua, err := normalize_user_agent(event.UserAgent)
		if err != nil {
			log.Printf("Unknown UA: %s", event.UserAgent)
			ua = "unknown"
		}

		datums[i*2] = types.MetricDatum{
			MetricName: aws.String("requests"),
			Timestamp:  &event.Time,
			Value:      aws.Float64(1),
			Unit:       types.StandardUnitCount,
			Dimensions: []types.Dimension{
				{
					Name:  aws.String("path"),
					Value: aws.String(event.Page),
				},
			},
		}

		datums[i*2+1] = types.MetricDatum{
			MetricName: aws.String("requests"),
			Timestamp:  &event.Time,
			Value:      aws.Float64(1),
			Unit:       types.StandardUnitCount,
			Dimensions: []types.Dimension{
				{
					Name:  aws.String("client"),
					Value: aws.String(ua),
				},
			},
		}
	}
	_, err := cw_client.PutMetricData(context.TODO(), &cloudwatch.PutMetricDataInput{
		Namespace:  aws.String("Blog"),
		MetricData: datums,
	})
	if err != nil {
		log.Fatal(err)
		return
	} else {
		fmt.Printf("Sent %d events to CW\n", len(events))
	}
}

func push_all_metrics_to_cloudwatch(cw_client *cloudwatch.Client, events chan metrics_record, done chan bool) {
	defer close(done)

	var wg sync.WaitGroup

	count := 0
	var to_push = make([]metrics_record, 0, 50)
	for metrics_event := range events {
		to_push = append(to_push, metrics_event)
		count += 1
		if count == 500 {
			wg.Add(1)
			go push_metrics_to_cloudwatch(&wg, cw_client, to_push)
			to_push = make([]metrics_record, 0, 50)
		}
	}
	wg.Add(1)
	go push_metrics_to_cloudwatch(&wg, cw_client, to_push)

	wg.Wait()
}

func handler(ctx context.Context, s3Event events.S3Event) {
	// Load the Shared AWS Configuration (~/.aws/config)
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithClientLogMode(aws.LogRequestWithBody))
	if err != nil {
		log.Fatal(err)
	}

	// Create an Amazon S3 service client
	s3_client := s3.NewFromConfig(cfg)
	cw_client := cloudwatch.NewFromConfig(cfg)
	ddb_client := dynamodb.NewFromConfig(cfg)

	keys := make([]string, 0)

	for _, record := range s3Event.Records {
		s3 := record.S3
		fmt.Printf("[%s - %s] Bucket = %s, Key = %s \n", record.EventSource, record.EventTime, s3.Bucket.Name, s3.Object.Key)
		if strings.HasPrefix(s3.Object.Key, "EO0JOSZAC367N") {
			keys = append(keys, s3.Object.Key)
			fmt.Printf("Record %s is part of CF logs\n", s3.Object.Key)
		} else {
			fmt.Printf("Record %s is not part of CF logs - ignoring\n", s3.Object.Key)
		}
	}

	metrics_batches := make(chan metrics_batch, len(keys))
	for _, key := range keys {
		go handle_record(s3_client, &key, metrics_batches)
	}

	events := make(chan metrics_record, 1000)

	metrics_done := make(chan bool)
	ddb_done := make(chan bool)
	go push_all_metrics_to_cloudwatch(cw_client, events, metrics_done)

	subscribers := make(map[string]int)

	for i := 0; i < len(keys); i++ {
		metrics_batch := <-metrics_batches
		fmt.Printf("Got a metrics batch with %d records\n", len(metrics_batch.metrics))
		for k, v := range metrics_batch.subscribers {
			subscribers[k] = v
		}

		for _, record := range metrics_batch.metrics {
			events <- record
		}
	}
	go push_subscribers_to_ddb(ddb_client, subscribers, ddb_done)
	close(events)
	_ = <-metrics_done
	_ = <-ddb_done
}

func main() {
	// Make the handler available for Remote Procedure Call by AWS Lambda
	lambda.Start(handler)
}
