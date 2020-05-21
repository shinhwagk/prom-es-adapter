package main

import (
	"encoding/json"
	"fmt"

	"github.com/olivere/elastic/v7"
)

func test() {
	query := elastic.NewBoolQuery()
	query = query.Filter(elastic.NewTermQuery("label.instance", "127.0.0.1"))
	dsl, err := query.Source()
	if err != nil {
		fmt.Println(err)
	}

	b, err := json.Marshal(dsl)
	if err != nil {
		fmt.Println("json.Marshal failed:", err)
		return
	}
	fmt.Println(string(b))
}
