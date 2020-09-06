package main

import (
	"errors"
	"flag"
	"fmt"
	production_system "production-system/pkg/production-system"
	"strings"
)

var (
	errFileNotFound = errors.New("file not found")
)

// A slice of facts names considered to be true
type trueFacts []string

func (trFacts *trueFacts) String() string {
	var strBuilder strings.Builder
	for _, trFact := range *trFacts {
		strBuilder.WriteString(trFact)
	}
	return strBuilder.String()
}

func (trFacts *trueFacts) Set(value string) error {
	if len(*trFacts) > 0 {
		return errors.New("true facts flag already have been set")
	}

	for _, fact := range strings.Split(value, ",") {
		*trFacts = append(*trFacts, fact)
	}

	return nil
}

func main() {
	var trueFacts trueFacts
	jsonFilePtr := flag.String("file", "test.json", "JSON file")
	flag.Var(&trueFacts, "facts", "A slice of facts names considered to be true")
	query := flag.String("query", "f1", "Fact that considered to be the query for the system")
	flag.Parse()

	engine, err := production_system.FromFile(*jsonFilePtr)
	if err != nil {
		fmt.Println(fmt.Sprintf("Error: %v", err))
		return
	}

	isDerived, usedRulesNames, err := engine.Forward(trueFacts, *query)
	if err != nil {
		fmt.Println(fmt.Sprintf("Error: %v", err))
		return
	}

	fmt.Printf("Is derived: %v, used rules: %v", isDerived, usedRulesNames)
}
