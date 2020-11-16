package production_system

// используемые пакеты
import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
)

// Задача:
// Написать продукционную систему, которая по заданным фактам и цели реализует вывод от цели и от данных
// Сама система получает исходные данные (список фактов и правил), а также задания для вывода
// (список имен истинных фактов имя факта-цели) из отдельных файлов формата .json

// При выводе от данных в цикле просматриваются все правила и выбираются те, все факты-входы которых
// полностью содеражтся в массиве истинных фактов. Факты-выход этих правил добавляется в массив истинных фактов.
// Цикл продолжается до тех пор, пока изменяется массив истинных фактов (добавляются новые факты-выходы).
// Если изменений не было, то определяется истинность факта-цели.
// Она определяется как этого факта-цели в массиве истинных фактов.

// При выводе от данных задача разбивается на подзадачи с помощью рекурсии:
// 	если подцель есть в списке истинных фактов, то нужно вернуть истину,
//	иначе для текущей подцели ищутся все правила из массива, факты-выходы которых совпадают с подцелью.
//	Факты-входы проверяются на наличие в массиве истинных фактов или выводимость (заход в рекурсию), если они выводимы,
//	то факт-выход добавляется в массив истинных фактов и просиходит выход из рекурсии.

// Rule является представлением правила
type Rule struct {
	Name         string  `json:"name"`
	Conditionals []*Fact `json:"conditionals"`
	Derivation   *Fact   `json:"derivation"`
}

// Fact фвляется представлением факта
type Fact struct {
	Name          string `json:"name"`
	SemanticValue string `json:"semantic_value"`
}

// Interpreter является представлением системы
type Interpreter struct {
	// Facts является множество известных фактов
	Facts map[string]*Fact `json:"facts"`
	// Rules является срезом (массивом) правил
	Rules []*Rule `json:"rules"`
}

// Вспомогательные классы для сериализации
type JSONRule struct {
	Name         string   `json:"name"`
	Conditionals []string `json:"conditionals"`
	Derivation   string   `json:"derivation"`
}

type JSONInterpreter struct {
	Facts []*Fact     `json:"facts"`
	Rules []*JSONRule `json:"rules"`
}

// in проверяет вхождение fact в срез facts
func in(facts []*Fact, fact *Fact) bool {
	for _, f := range facts {
		if f == fact {
			return true
		}
	}

	return false
}

// _fromJSONEngine проверяет содержимое, полученное из файла
func _fromJSONEngine(jsonEngine *JSONInterpreter) (*Interpreter, error) {
	engine := new(Interpreter)
	engine.Facts = make(map[string]*Fact, 0)
	engine.Rules = make([]*Rule, 0)

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
			conditionals = append(conditionals, engine.Facts[conditional])
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

// FromFile загружает Interpreter из файла
func FromFile(filepath string) (*Interpreter, error) {
	jsonFile, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}

	defer jsonFile.Close()
	jsonBytes, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	var jsonEngine JSONInterpreter
	err = json.Unmarshal(jsonBytes, &jsonEngine)
	if err != nil {
		return nil, err
	}

	return _fromJSONEngine(&jsonEngine)
}

// forward вспомогательный метод для поиска от данных
func (e *Interpreter) forward(trueFacts []*Fact, query *Fact) (bool, []*Rule) {
	// срез использованных правил
	usedRules := make([]*Rule, 0)
	// бесконечный цикл
	for {
		// срез новых истинных фактов
		resultFacts := make([]*Fact, 0)
		// перебираем все правила
		for _, rule := range e.Rules {
			matches := 0
			derived := true
			// перебираем все истинные факты
			for _, trFact := range trueFacts {
				// если у правила вывод уже истинен, то пропускаем его
				if rule.Derivation == trFact {
					derived = false
				} else {
					// иначе проверяем, что факты-условия правила находятся в списке истинных фактов
					for _, conditional := range rule.Conditionals {
						if conditional == trFact {
							matches++
						}
					}
				}
			}
			// если правило доказуемо
			if derived && matches == len(rule.Conditionals) {
				// обновляем срезы с данными
				resultFacts = append(resultFacts, rule.Derivation)
				usedRules = append(usedRules, rule)
			}
		}
		// обновляем срез с истинными фактами
		for _, resultFact := range resultFacts {
			trueFacts = append(trueFacts, resultFact)
		}
		// если не получили новых данных (перебрали все возможные варианты)
		if len(resultFacts) == 0 {
			// проверяем, находится ли цель в срезе с истинными фактами
			return in(trueFacts, query), usedRules
		}
	}
}

// _isDerivable проверяет подцель fact на выводимость
func (e *Interpreter) _isDerivable(trueFacts []*Fact, fact *Fact, usedRules []*Rule) bool {
	// если подцель уже в срезе истинных фактов
	if in(trueFacts, fact) {
		return true
	}

	// проверяем подцель на выводимость через правила
	for _, rule := range e.Rules {
		// вывод правила совпадает с текущей подцелью
		if rule.Derivation == fact {
			// счетчик истинных фактов-условий
			derivableCount := 0

			// проверяем на истинность или выводимость каждый факт-условие из этого правила
			for _, conditional := range rule.Conditionals {
				if in(trueFacts, conditional) && e._isDerivable(trueFacts, conditional, usedRules) {
					derivableCount++
				}
			}

			// проверяем, что все факты-условия правила истины
			if derivableCount == len(rule.Conditionals) {
				// обновляем срезы с данными
				// добвляем это правило в использованные
				usedRules = append(usedRules, rule)
				// добавляем вывод правила к истинным фактам
				trueFacts = append(trueFacts, rule.Derivation)
				return true
			}
		}
	}

	return false
}

// backward вспомогательный метод для поиска от цели
func (e *Interpreter) backward(trueFacts []*Fact, query *Fact) (bool, []*Rule) {
	// срез для использованных правил
	usedRules := make([]*Rule, 0)
	// начинаем разбор от заданной цели
	isDerived := e._isDerivable(trueFacts, query, usedRules)
	return isDerived, usedRules
}

// _convertNames преобразует имена фактов и цели в объекты структуры Fact
func (e *Interpreter) _convertNames(trueFactNames []string, queryName string) ([]*Fact, *Fact, error) {
	trueFacts := make([]*Fact, 0)

	for _, trFactName := range trueFactNames {
		// проверяем, что имя факта содержится в памяти
		if fact, exists := e.Facts[trFactName]; exists {
			trueFacts = append(trueFacts, fact)
		} else {
			// иначе возвращаем ошибку
			return make([]*Fact, 0), nil, errors.New(fmt.Sprintf("Unknown fact: %+v", trFactName))
		}
	}

	queryFact, exists := e.Facts[queryName]
	// проверяем, что цель содержится во множестве известных фактов
	if !exists {
		return make([]*Fact, 0), nil, errors.New(fmt.Sprintf("Unknown fact: %+v", queryName))
	}

	return trueFacts, queryFact, nil
}

// Forward проверяет цели с именем queryName и исходными данными trueFactNames через выводимость от данных
func (e *Interpreter) Forward(trueFactNames []string, queryName string) (bool, []string, error) {
	trueFacts, queryFact, err := e._convertNames(trueFactNames, queryName)
	if err != nil {
		return false, nil, err
	}
	isDerived, usedRules := e.forward(trueFacts, queryFact)
	// срез с именами использованных правил
	usedRulesNames := make([]string, 0)
	for _, rule := range usedRules {
		usedRulesNames = append(usedRulesNames, rule.Name)
	}

	return isDerived, usedRulesNames, nil
}

// Backward проверяет с именем queryName и исходными данными trueFactNames выводимость от цели
func (e *Interpreter) Backward(trueFactNames []string, queryName string) (bool, []string, error) {
	// переводим строки с именами фактов в объекты
	trueFacts, queryFact, err := e._convertNames(trueFactNames, queryName)
	// возвращаем ошибку при переводе
	if err != nil {
		return false, nil, err
	}
	// вызываем метод вывода
	isDerived, usedRules := e.backward(trueFacts, queryFact)
	// срез имен использованных правил
	usedRulesNames := make([]string, 0)
	for _, rule := range usedRules {
		usedRulesNames = append(usedRulesNames, rule.Name)
	}

	return isDerived, usedRulesNames, nil
}
