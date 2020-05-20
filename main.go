package main

import (
	"fmt"
	"log"
)

// func merge1(ms ...map[string]interface{}) map[string]interface{} {
// 	O := map[string]interface{}{}
// 	for _, i := range ms {
// 		mergo.Merge(&O, i)
// 	}
// 	return O
// }

// package main

// import _ "github.com/imdario/mergo"

type BoolQuery struct {
	Query
	mustClauses        []Query
	mustNotClauses     []Query
	filterClauses      []Query
	shouldClauses      []Query
	boost              *float64
	minimumShouldMatch string
	adjustPureNegative *bool
	queryName          string
}

func (q *BoolQuery) MustNot(queries ...Query) *BoolQuery {
	q.mustNotClauses = append(q.mustNotClauses, queries...)
	return q
}

func (q *BoolQuery) Filter(filters ...Query) *BoolQuery {
	q.filterClauses = append(q.filterClauses, filters...)
	return q
}

func (q *TermQuery) Source() string {
	// {"term":{"name":"value"}}
	// source := make(map[string]interface{})
	// tq := make(map[string]interface{})
	// source["term"] = tq

	// if q.boost == nil && q.queryName == "" {
	// 	tq[q.name] = q.value
	// } else {
	// 	subQ := make(map[string]interface{})
	// 	subQ["value"] = q.value
	// 	if q.boost != nil {
	// 		subQ["boost"] = *q.boost
	// 	}
	// 	if q.queryName != "" {
	// 		subQ["_name"] = q.queryName
	// 	}
	// 	tq[q.name] = subQ
	// }
	return "1"
}

type Query interface {
	// Source returns the JSON-serializable query request.
	// Source() (interface{}, error)
	Source() string
}

type TermQuery struct {
	name      string
	value     interface{}
	boost     *float64
	queryName string
}
type RegexpQuery struct {
	name                  string
	regexp                string
	flags                 string
	boost                 *float64
	rewrite               string
	queryName             string
	maxDeterminizedStates *int
}

func NewBoolQuery() *BoolQuery {
	return &BoolQuery{
		mustClauses:    make([]Query, 0),
		mustNotClauses: make([]Query, 0),
		filterClauses:  make([]Query, 0),
	}
}

func NewTermQuery(name string, value interface{}) *TermQuery {
	return &TermQuery{name: name, value: value}
}

func NewRegexpQuery(name string, regexp string) *RegexpQuery {
	return &RegexpQuery{name: name, regexp: regexp}
}

func main() {
	log.Println(`started`)
	// O := map[string]interface{}{}

	// A := map[string]interface{}{
	// 	"a": map[string]interface{}{
	// 		"a": "b",
	// 		"c": 11,
	// 	},
	// }
	// B := map[string]interface{}{
	// 	"a": map[string]interface{}{
	// 		"b": "b",
	// 	},
	// }

	// X := merge1(A, B)
	// fmt.Println(X)

	query := NewBoolQuery()
	query = query.Filter(NewTermQuery("label.NAME", "A"))
	fmt.Println(len(query.filterClauses))

}
