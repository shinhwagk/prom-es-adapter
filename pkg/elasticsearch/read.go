package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"

	goelastic "github.com/elastic/go-elasticsearch/v7"
	elastic "github.com/olivere/elastic/v7"
	"github.com/prometheus/prometheus/prompb"
	"go.uber.org/zap"
)

type ReadService struct {
	client *goelastic.Client
	config *ReadConfig
	logger *zap.Logger
}

// ReadConfig configures the ReadService
type ReadConfig struct {
	Alias   string
	MaxDocs int
}

// NewReadService will create a new ReadService
func NewReadService(logger *zap.Logger, client *goelastic.Client, config *ReadConfig) *ReadService {
	svc := &ReadService{
		client: client,
		config: config,
		logger: logger,
	}
	// TODO: add stats
	return svc
}

// Read will perform Elasticsearch query
func (svc *ReadService) Read(ctx context.Context, req []*prompb.Query) ([]*prompb.QueryResult, error) {
	results := make([]*prompb.QueryResult, 0, len(req))
	for _, q := range req {
		resp := svc.buildCommand(q)
		ts, err := svc.createTimeseries(resp.Hits)
		if err != nil {
			return nil, err
		}
		results = append(results, &prompb.QueryResult{Timeseries: ts})
	}
	return results, nil
}

type EsQUERY struct {
	query interface{}
}

func (svc *ReadService) buildCommand(q *prompb.Query) *elastic.SearchResult {
	query := elastic.NewBoolQuery()
	for _, m := range q.Matchers {
		switch m.Type {
		case prompb.LabelMatcher_EQ:
			query = query.Filter(elastic.NewTermQuery("label."+m.Name, m.Value))
		case prompb.LabelMatcher_NEQ:
			query = query.MustNot(elastic.NewTermQuery("label."+m.Name, m.Value))
		case prompb.LabelMatcher_RE:
			query = query.Filter(elastic.NewRegexpQuery("label."+m.Name, m.Value))
		case prompb.LabelMatcher_NRE:
			query = query.MustNot(elastic.NewRegexpQuery("label."+m.Name, m.Value))
		default:
			svc.logger.Panic("unknown match", zap.String("type", m.Type.String()))
		}
	}

	query = query.Filter(elastic.NewRangeQuery("timestamp").Gte(q.StartTimestampMs).Lte(q.EndTimestampMs))
	inf, err := query.Source()
	if err != nil {
		log.Fatalf("Error encoding query: %s", err)
	}
	esquery := EsQUERY{query: inf}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(esquery); err != nil {
		log.Fatalf("Error encoding query: %s", err)
	}
	fmt.Println(inf, buf.String())
	res, err := svc.client.Search(
		svc.client.Search.WithContext(context.Background()),
		svc.client.Search.WithIndex(svc.config.Alias+"-*"),
		svc.client.Search.WithBody(&buf),
		svc.client.Search.WithTrackTotalHits(true),
		svc.client.Search.WithPretty(),
	)
	if err != nil {
		svc.logger.Fatal("Error getting response: %s", zap.Error(err))
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			log.Fatalf("Error parsing the response body: %s", err)
		} else {
			log.Fatalf("[%s] %s: %s",
				res.Status(),
				e["error"].(map[string]interface{})["type"],
				e["error"].(map[string]interface{})["reason"],
			)
		}
	}
	var r elastic.SearchResult
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		log.Fatalf("Error parsing the response body: %s", err)
	}
	return &r
}

func (svc *ReadService) createTimeseries(results *elastic.SearchHits) ([]*prompb.TimeSeries, error) {
	tsMap := make(map[string]*prompb.TimeSeries)
	for _, r := range results.Hits {
		var s prometheusSample
		if err := json.Unmarshal(r.Source, &s); err != nil {
			svc.logger.Fatal("Failed to unmarshal sample", zap.Error(err))
		}
		fingerprint := s.Labels.Fingerprint().String()

		ts, ok := tsMap[fingerprint]
		if !ok {
			labels := make([]*prompb.Label, 0, len(s.Labels))
			for k, v := range s.Labels {
				labels = append(labels, &prompb.Label{
					Name:  string(k),
					Value: string(v),
				})
			}
			ts = &prompb.TimeSeries{
				Labels: labels,
			}
			tsMap[fingerprint] = ts
		}
		ts.Samples = append(ts.Samples, prompb.Sample{
			Value:     s.Value,
			Timestamp: s.Timestamp,
		})
	}
	ret := make([]*prompb.TimeSeries, 0, len(tsMap))

	for _, s := range tsMap {
		ret = append(ret, s)
	}
	return ret, nil
}
