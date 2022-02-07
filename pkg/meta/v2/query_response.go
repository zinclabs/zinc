package v2

import "time"

// SearchResponse for a query
type SearchResponse struct {
	Took         int                            `json:"took"` // Time it took to generate the response
	TimedOut     bool                           `json:"timed_out"`
	Hits         Hits                           `json:"hits"`
	Aggregations map[string]AggregationResponse `json:"aggregations,omitempty"`
	Error        string                         `json:"error"`
}

type Hits struct {
	Total    Total   `json:"total"`
	MaxScore float64 `json:"max_score"`
	Hits     []Hit   `json:"hits"`
}

type Hit struct {
	Index     string      `json:"_index"`
	Type      string      `json:"_type"`
	ID        string      `json:"_id"`
	Score     float64     `json:"_score"`
	Timestamp time.Time   `json:"@timestamp"`
	Source    interface{} `json:"_source"`
}

type Total struct {
	Value int `json:"value"` // Count of documents returned
}

type AggregationResponse struct {
	Value        interface{}                  `json:"value,omitempty"`
	Buckets      []AggregationBucket          `json:"buckets,omitempty"`
	NamedBuckets map[string]AggregationBucket `json:"buckets,omitempty"`
}

type AggregationBucket struct {
	Key          interface{}                    `json:"key"`
	KeyAsString  string                         `json:"key_as_string,omitempty"`
	DocCount     uint64                         `json:"doc_count"`
	Aggregations map[string]AggregationResponse `json:"aggregations,omitempty"`
}
