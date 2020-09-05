package production_system

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
)

type Rule struct {
	Name         string  `json:"name"`
	Conditionals []*Fact `json:"conditionals"`
	Derivation   *Fact   `json:"derivation"`
}

type Fact struct {
	Name          string `json:"name"`
	SemanticValue string `json:"semantic_value"`
}

type Engine struct {
	Facts map[string]*Fact `json:"facts"`
	Rules []*Rule          `json:"rules"`
}

type JSONRule struct {
	Name         string   `json:"name"`
	Conditionals []string `json:"conditionals"`
	Derivation   string   `json:"derivation"`
}

type JSONEngine struct {
	Facts []*Fact     `json:"facts"`
	Rules []*JSONRule `json:"rules"`
}

func in(facts []*Fact, fact *Fact) bool {
	for _, f := range facts {
		if f == fact {
			return true
		}
	}

	return false
}

func _fromJSONEngine(jsonEngine *JSONEngine) (*Engine, error) {
	engine := new(Engine)

	for _, fact := range jsonEngine.Facts {
		if _, known := engine.Facts[fact.Name]; known {
			return nil, errors.New(fmt.Sprintf("Doubled fact: %+v", fact))
		}
		engine.Facts[fact.Name] = fact
	}

	knownRules := make(map[string]struct{}, 0)

	for _, jrule := range jsonEngine.Rules {
		conditionals := make([]*Fact, 0)
		if _, known := knownRules[jrule.Name]; known {
			return nil, errors.New(fmt.Sprintf("Doubled rule: %v", jrule))
		}

		for _, conditional := range jrule.Conditionals {
			if _, exists := engine.Facts[conditional]; !exists {
				return nil, errors.New(fmt.Sprintf("Unknown fact name %v in rule %v", conditional, jrule))
			}
			updated := append(conditionals, engine.Facts[conditional])
			conditionals = updated
		}

		if _, exists := engine.Facts[jrule.Derivation]; !exists {
			return nil, errors.New(fmt.Sprintf("Unknown fact name %v in rule %v", jrule.Derivation, jrule))
		}

		rule := new(Rule)
		rule.Name = jrule.Name
		rule.Conditionals = conditionals
		rule.Derivation = engine.Facts[jrule.Derivation]

		updated := append(engine.Rules, rule)
		engine.Rules = updated
	}

	return engine, nil
}

func FromFile(filepath string) (*Engine, error) {
	jsonFile, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}

	defer jsonFile.Close()
	jsonBytes, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	var jsonEngine JSONEngine
	err = json.Unmarshal(jsonBytes, &jsonEngine)
	if err != nil {
		return nil, err
	}

	return _fromJSONEngine(&jsonEngine)
}

func (e *Engine) forward(trueFacts []*Fact, query *Fact) (bool, []*Rule) {
	for {
		resultFacts := make([]*Fact, 0)
		usedRules := make([]*Rule, 0)

		for _, rule := range e.Rules {
			matches := 0
			derived := true

			for _, trFact := range trueFacts {
				if rule.Derivation == trFact {
					derived = false
				} else {
					for _, conditional := range rule.Conditionals {
						if conditional == trFact {
							matches++
						}
					}
				}
			}

			if matches == len(rule.Conditionals) && derived {
				updated := append(resultFacts, rule.Derivation)
				resultFacts = updated

				up := append(usedRules, rule)
				usedRules = up
			}
		}

		for _, resultFact := range resultFacts {
			updated := append(trueFacts, resultFact)
			trueFacts = updated
		}

		if len(resultFacts) == 0 {
			return in(trueFacts, query), usedRules
		}
	}
}

func (e *Engine) _isDerivable(trueFacts []*Fact, fact *Fact, usedRules []*Rule) bool {
	if in(trueFacts, fact) {
		return true
	}

	for _, rule := range e.Rules {
		if rule.Derivation == fact {
			derivableCount := 0

			for _, conditional := range rule.Conditionals {
				if e._isDerivable(trueFacts, conditional, usedRules) && in(trueFacts, conditional) {
					derivableCount++
				}
			}

			if derivableCount == len(rule.Conditionals) {
				updated := append(usedRules, rule)
				usedRules = updated

				up := append(trueFacts, rule.Derivation)
				trueFacts = up

				return true
			}
		}
	}

	return false
}

func (e *Engine) backward(trueFacts []*Fact, query *Fact) (bool, []*Rule) {
	usedRules := make([]*Rule, 0)
	isDerived := e._isDerivable(trueFacts, query, usedRules)

	return isDerived, usedRules
}

func (e *Engine) _convertNames(trueFactNames []string, queryName string) ([]*Fact, *Fact, error) {
	trueFacts := make([]*Fact, 0)

	for _, trFactName := range trueFactNames {
		if fact, exists := e.Facts[trFactName]; exists {
			updated := append(trueFacts, fact)
			trueFacts = updated
		} else {
			return make([]*Fact, 0), nil, errors.New(fmt.Sprintf("Unknown fact: %+v", trueFactNames))
		}
	}

	queryFact, exists := e.Facts[queryName]
	if !exists {
		return make([]*Fact, 0), nil, errors.New(fmt.Sprintf("Unknown fact: %+v", queryName))
	}

	return trueFacts, queryFact, nil
}

func (e *Engine) Forward(trueFactNames []string, queryName string) (bool, []string, error) {
	trueFacts, queryFact, err := e._convertNames(trueFactNames, queryName)
	if err != nil {
		return false, nil, err
	}
	isDerived, usedRules := e.forward(trueFacts, queryFact)

	usedRulesNames := make([]string, 0)

	for _, rule := range usedRules {
		usedRulesNames = append(usedRulesNames, rule.Name)
	}

	return isDerived, usedRulesNames, nil
}

func (e *Engine) Backward(trueFactNames []string, queryName string) (bool, []*Rule, error) {
	trueFacts, queryFact, err := e._convertNames(trueFactNames, queryName)
	if err != nil {
		return false, nil, err
	}

	isDerived, usedRules := e.backward(trueFacts, queryFact)
	return isDerived, usedRules, nil
}
