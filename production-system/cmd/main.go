package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	production_system "production-system/pkg/production-system"
)

var (
	errFileNotFound = errors.New("file not found")
)

type Task struct {
	TrueFacts []string `json:"true_facts"`
	Query     string   `json:"query"`
}

func FromFile(filepath string) (*Task, error) {
	jsonFile, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}

	defer jsonFile.Close()
	jsonBytes, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}
	task := new(Task)
	err = json.Unmarshal(jsonBytes, &task)
	if err != nil {
		return nil, err
	}

	return task, nil

}

func main() {
	jsonEngineFilePtr := flag.String("f", "test.json", "JSON file")
	jsonForwardPtr := flag.String("forward", "forward.json", "")
	jsonBackwardPtr := flag.String("backward", "backward.json", "")

	flag.Parse()

	engine, err := production_system.FromFile(*jsonEngineFilePtr)
	if err != nil {
		fmt.Println(fmt.Sprintf("Error: %v", err))
		return
	}

	forwardTask, err := FromFile(*jsonForwardPtr)
	if err != nil {
		fmt.Println(fmt.Sprintf("Error: %v", err))
		return
	}

	isDerived, usedRulesNames, err := engine.Forward(forwardTask.TrueFacts, forwardTask.Query)
	if err != nil {
		fmt.Println(fmt.Sprintf("Error: %v", err))
		return
	}

	fmt.Println(fmt.Sprintf("[forward] :: Is derived: %v, used rules: %v", isDerived, usedRulesNames))

	backwardTask, err := FromFile(*jsonBackwardPtr)
	if err != nil {
		fmt.Println(fmt.Sprintf("Error: %v", err))
		return
	}

	isDerived, usedRulesNames, err = engine.Forward(backwardTask.TrueFacts, backwardTask.Query)
	if err != nil {
		fmt.Println(fmt.Sprintf("Error: %v", err))
		return
	}

	fmt.Println(fmt.Sprintf("[backward] :: Is derived: %v, used rules: %v", isDerived, usedRulesNames))
}
