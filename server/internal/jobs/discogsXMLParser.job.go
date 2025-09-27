package jobs

import (
	"context"
	"waugzee/internal/logger"
	"waugzee/internal/services"
)

type DiscogsXMLParserJob struct {
	xmlParser *services.DiscogsXMLParserService
	log       logger.Logger
	schedule  services.Schedule
}

func NewDiscogsXMLParserJob(
	xmlParser *services.DiscogsXMLParserService,
	schedule services.Schedule,
) *DiscogsXMLParserJob {
	log := logger.New("discogsXMLParserJob")
	log.Info("Creating new Discogs XML parser job", "schedule", schedule)

	return &DiscogsXMLParserJob{
		xmlParser: xmlParser,
		log:       log,
		schedule:  schedule,
	}
}

func (j *DiscogsXMLParserJob) Name() string {
	return "DiscogsXMLParser"
}

func (j *DiscogsXMLParserJob) Execute(ctx context.Context) error {
	log := j.log.Function("Execute")

	if err := j.xmlParser.ParseXMLFiles(ctx); err != nil {
		return log.Err("XML parsing failed", err)
	}

	log.Info("Discogs XML parser job completed successfully")
	return nil
}

func (j *DiscogsXMLParserJob) Schedule() services.Schedule {
	return j.schedule
}

