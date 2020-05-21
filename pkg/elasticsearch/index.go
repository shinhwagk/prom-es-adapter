package elasticsearch

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"time"

	elastic "github.com/olivere/elastic/v7"
	"go.uber.org/zap"
)

// IndexService will manage alias and indexes derived from the configured index alias

// IndexService will manage alias and indexes derived from the configured index alias
type IndexService struct {
	ctx    context.Context
	client *elastic.Client
	config *IndexConfig
	logger *zap.Logger
}

// IndexConfig is used to configure IndexService
type IndexConfig struct {
	Alias   string
	MaxAge  string
	MaxDocs int64
	MaxSize string
}

// IndexTemplateConfig is used to resolve template
type IndexTemplateConfig struct {
	Alias    string
	Shards   int
	Replicas int
}

// NewIndexService will ensure required alias and indexes exist.  It will also monitor
// active index and rollover as necessary
func NewIndexService(ctx context.Context, logger *zap.Logger, client *elastic.Client, config *IndexConfig) (*IndexService, error) {
	svc := &IndexService{
		ctx:    ctx,
		client: client,
		config: config,
		logger: logger,
	}
	if err := svc.createIndex(); err != nil {
		return nil, err
	}
	go svc.rolloverIndex()
	return svc, nil
}

func EnsureIndexTemplate(ctx context.Context, client *elastic.Client, config *IndexTemplateConfig) error {
	var buf bytes.Buffer
	t := template.Must(template.New("template").Parse(indexTemplate))
	err := t.Execute(&buf, config)
	if err != nil {
		return fmt.Errorf("executing template: %s", err)
	}
	payload := buf.String()

	_, err = client.IndexPutTemplate(config.Alias).BodyString(payload).Do(ctx)
	if err != nil {
		return fmt.Errorf("Failed to create index template: %s", err)
	}
	return nil
}

func (svc *IndexService) createIndex() error {
	exists, err := svc.client.IndexExists(svc.config.Alias).Do(svc.ctx)
	if err != nil {
		return err
	}
	if !exists {
		var buf bytes.Buffer
		t := template.Must(template.New("create").Parse(indexCreate))
		err := t.Execute(&buf, svc.config)
		if err != nil {
			return fmt.Errorf("executing template: %s", err)
		}
		payload := buf.String()

		_, err = svc.client.CreateIndex(svc.config.Alias + "-1").BodyString(payload).Do(svc.ctx)
		if err != nil {
			return fmt.Errorf("Failed to create initial index: %s", err)
		}
	}
	return nil
}

// rolloverIndex
func (svc *IndexService) rolloverIndex() error {
	rollover := svc.client.RolloverIndex(svc.config.Alias)
	if svc.config.MaxAge != "" {
		rollover.AddMaxIndexAgeCondition(svc.config.MaxAge)
	}
	if svc.config.MaxDocs > 0 {
		rollover.AddMaxIndexDocsCondition(svc.config.MaxDocs)
	}
	if svc.config.MaxSize != "" {
		rollover.AddCondition("max_size", svc.config.MaxSize)
	}
	for {
		select {
		case <-time.After(5 * time.Minute):
			res, err := rollover.Do(svc.ctx)
			if err != nil {
				svc.logger.Error("Failed to rollover index", zap.Error(err))
			} else {
				svc.logger.Debug(fmt.Sprintf("%+v", res))
			}
		case <-svc.ctx.Done():
			svc.logger.Info("Index service exiting")
			return svc.ctx.Err()
		}
	}
}
