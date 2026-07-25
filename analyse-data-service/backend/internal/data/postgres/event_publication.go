package postgres

import (
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventpublication"
)

type EventPublicationStore = eventpublication.Store
type EventPublicationTransaction = eventpublication.Transaction
type PublicationRawDocument = eventpublication.PublicationRawDocument
type PublicationEvent = eventpublication.PublicationEvent
type PublicationEventSource = eventpublication.PublicationEventSource
type PublicationEventTag = eventpublication.PublicationEventTag
type PublicationCollectorExecution = eventpublication.PublicationCollectorExecution
type PublicationReviewMetadata = eventpublication.PublicationReviewMetadata
type PublicationWriteCounts = eventpublication.PublicationWriteCounts
type EventPublicationReceipt = eventpublication.EventPublicationReceipt
