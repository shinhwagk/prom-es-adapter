package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"

	goelastic "github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esutil"
	elastic "github.com/olivere/elastic/v7"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"
	"go.uber.org/zap"
)

type prometheusSample struct {
	Labels    model.Metric `json:"label"`
	Value     float64      `json:"value"`
	Timestamp int64        `json:"timestamp"`
}

// WriteService will proxy Prometheus write requests to Elasticsearch
type WriteService struct {
	config   *WriteConfig
	logger   *zap.Logger
	esClient *goelastic.Client
}

// WriteConfig is used to configure WriteService
type WriteConfig struct {
	Alias   string
	Daily   bool
	MaxAge  int
	MaxDocs int
	MaxSize int
	Workers int
	Stats   bool
}

// NewWriteService creates and returns a new elasticsearch WriteService
func NewWriteService(ctx context.Context, logger *zap.Logger, client *goelastic.Client) (*WriteService, error) {
	svc := &WriteService{
		logger:   logger,
		esClient: client,
	}

	// prometheus.MustRegister(svc)
	return svc, nil
}

// Close will close the underlying elasticsearch BulkProcessor
func (svc *WriteService) Close() error {
	// return svc.processor.Close()
	return nil
}

// Write will enqueue Prometheus sample data to be batch written to Elasticsearch
func (svc *WriteService) Write(req []*prompb.TimeSeries) {
	test(svc, req)
	test(svc, req)
	test(svc, req)
	test(svc, req)
	test(svc, req)
}

func (svc *WriteService) after(id int64, requests []elastic.BulkableRequest, response *elastic.BulkResponse, err error) {
	if err != nil {
		svc.logger.Error(err.Error())
	} else {
		for _, i := range response.Items {
			if i["index"].Status != 201 {
				svc.logger.Error(fmt.Sprintf("%+v", i["index"].Error))
			}
		}
	}
}

func test(svc *WriteService, req []*prompb.TimeSeries) {
	index := "prom-metrics-1"
	indexer, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client:     svc.esClient,
		Index:      index,
		NumWorkers: 1,
	})
	if err != nil {
		log.Fatalf("Error creating the indexer: %s", err)
	}
	for _, ts := range req {
		metric := make(model.Metric, len(ts.Labels))
		for _, l := range ts.Labels {
			metric[model.LabelName(l.Name)] = model.LabelValue(l.Value)
		}
		for _, s := range ts.Samples {
			v := float64(s.Value)
			if math.IsNaN(v) || math.IsInf(v, 0) {
				svc.logger.Debug(fmt.Sprintf("invalid value %+v, skipping sample %+v", v, s))
				continue
			}
			sample := prometheusSample{
				metric,
				v,
				s.Timestamp,
			}
			var buf bytes.Buffer
			if err := json.NewEncoder(&buf).Encode(sample); err != nil {
				log.Fatalf("Error encoding query: %s", err)
			}
			err := indexer.Add(context.Background(),
				esutil.BulkIndexerItem{
					Action: "index",
					Body:   strings.NewReader(buf.String()),
					OnSuccess: func(
						ctx context.Context,
						item esutil.BulkIndexerItem,
						res esutil.BulkIndexerResponseItem,
					) {
						// fmt.Printf("[%d] %s test/%s", res.Status, res.Result, item.DocumentID)
					},
					OnFailure: func(
						ctx context.Context,
						item esutil.BulkIndexerItem,
						res esutil.BulkIndexerResponseItem, err error,
					) {
						if err != nil {
							log.Printf("ERROR: %s", err)
						} else {
							log.Printf("ERROR: %s: %s", res.Error.Type, res.Error.Reason)
						}
					},
				},
			)
			if err != nil {
				log.Fatalf("Unexpected error: %s", err)
			}
		}
	}
	if err := indexer.Close(context.Background()); err != nil {
		log.Fatalf("Unexpected error: %s", err)
	}
	stats := indexer.Stats()
	if stats.NumFailed > 0 {
		log.Fatalf("Indexed [%d] documents with [%d] errors", stats.NumFlushed, stats.NumFailed)
	} else {
		log.Printf("Successfully indexed [%d] documents", stats.NumFlushed)
	}
}
