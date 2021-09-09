package main

import (
	"fmt"
)

type MatchersEmitter struct {
	AST              map[string]*Node
	resolvedMatchers map[string]interface{}
	currentState     map[interface{}]interface{}
}

type filterArgument struct {
	Number *int    `json:"num,omitempty"`
	String *string `json:"string,omitempty"`
}

func (a filterArgument) equal(valueToCompare interface{}) bool {
	if a.Number != nil {
		intToCompare, ok := valueToCompare.(int)
		return ok && intToCompare == *a.Number
	}
	if a.String != nil {
		stringToCompare, ok := valueToCompare.(string)
		return ok && stringToCompare == *a.String
	}
	return false
}

func filter(input interface{}, path []Step, argument *filterArgument) (interface{}, error) {
	step := path[0]
	if step.Identifier != nil && step.Index != nil {
		return nil, fmt.Errorf("bad path expression ast: step cannot contain contain index and identifier")
	} else if step.Identifier == nil && step.Index == nil {
		return nil, fmt.Errorf("bad path expression ast: step has contain index or identifier")
	}
	if len(path) > 1 {
		nextPath := path[1:len(path)]
		// Walk recursively the rest of the path
		if step.Identifier != nil {
			filteredInput, err := filter(input.(map[interface{}]interface{})[*step.Identifier], nextPath, argument)
			if err != nil {
				return nil, err
			}
			return map[interface{}]interface{}{*step.Identifier: filteredInput}, nil
		}
		if step.Index != nil {
			filteredInput, err := filter(input.([]interface{})[*step.Index], nextPath, argument)
			if err != nil {
				return nil, err
			}
			return []interface{}{filteredInput}, nil
		}
	} else {
		if argument != nil {
			// This is the end, filter the field with the argument
			if step.Identifier == nil {
				return nil, fmt.Errorf("bad path expression ast: terminal step only accepts identifier")
			}
			filteredSlice := []interface{}{}
			for _, inputItem := range input.([]interface{}) {
				mapToFilter := inputItem.(map[interface{}]interface{})
				valueToCompare, ok := mapToFilter[*step.Identifier]
				if !ok {
					return nil, fmt.Errorf("bad path expression: %s does not exist", *step.Identifier)
				}
				if argument == nil {
					return nil, fmt.Errorf("TODO: filter without argument is not supported")
				}
				if argument.equal(valueToCompare) {
					filteredSlice = append(filteredSlice, inputItem)
				}
			}
			return filteredSlice, nil
		} else {
			return input, nil
		}
	}
	return nil, nil
}

func (e MatchersEmitter) resolveMatchersPath(state interface{}, path []Step) ([]Step, interface{}, error) {
	if path[0].Identifier != nil && *path[0].Identifier == "matchers" {
		result, err := e.resolveMatcher(state, *path[1].Identifier)
		if err != nil {
			return nil, nil, err
		}
		path := path[2:len(path)]
		return path, result, nil
	}
	return path, state, nil
}

func (e MatchersEmitter) resolvePath(state interface{}, path []Step) (interface{}, error) {
	path, result, err := e.resolveMatchersPath(state, path)
	if err != nil {
		return nil, err
	}
	// Use the resolved matcher as the input
	for _, step := range path {
		if step.Identifier != nil {
			result = result.(map[interface{}]interface{})[*step.Identifier]
		} else if step.Index != nil {
			result = result.([]interface{})[*step.Index]
		}
	}
	return result, nil
}

func (e MatchersEmitter) resolveFilterArgument(state interface{}, argument Argument) (*filterArgument, error) {
	if argument.Path != nil {
		result, err := e.resolvePath(state, argument.Path)
		if err != nil {
			return nil, err
		}
		stringResult, ok := result.(string)
		if ok {
			return &filterArgument{String: &stringResult}, nil
		}
		intResult, ok := result.(int)
		if ok {
			return &filterArgument{Number: &intResult}, nil
		}
		return nil, fmt.Errorf("TODO: path result %+v not supported", result)
	}
	return &filterArgument{argument.Number, argument.String}, nil
}

func (e MatchersEmitter) resolveFilter(state interface{}, arguments []Argument) (interface{}, error) {
	lhs := arguments[0]
	if lhs.Path == nil {
		return nil, fmt.Errorf("bad filter expression: left argument should be a path")
	}

	var argument *filterArgument
	if len(arguments) > 1 {
		// is a filter with comparation
		rhs := arguments[1]
		var err error
		argument, err = e.resolveFilterArgument(state, rhs)
		if err != nil {
			return nil, err
		}
	}

	lhsPath, currentState, err := e.resolveMatchersPath(state, lhs.Path)
	if err != nil {
		return nil, err
	}
	result, err := filter(currentState, lhsPath, argument)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (e MatchersEmitter) resolveNode(state interface{}, node *Node) (interface{}, error) {
	resolvedNode := state
	switch node.Expression.Operator {
	case 0:
	case Equal:
		var err error
		resolvedNode, err = e.resolveFilter(state, node.Expression.Arguments)
		if err != nil {
			return nil, err
		}
	}
	if node.Pipe != nil {
		var err error
		resolvedNode, err = e.resolveNode(resolvedNode, node.Pipe)
		if err != nil {
			return nil, err
		}
	}
	return resolvedNode, nil
}

func (e MatchersEmitter) resolveMatcher(state interface{}, name string) (interface{}, error) {

	resolvedMatcher, ok := e.resolvedMatchers[name]
	if ok {
		return resolvedMatcher, nil
	}
	node := e.AST[name]
	var err error
	resolvedMatcher, err = e.resolveNode(state, node)
	if err != nil {
		return nil, err
	}
	e.resolvedMatchers[name] = resolvedMatcher
	return resolvedMatcher, nil
}

func (e MatchersEmitter) Emit() (map[string]interface{}, error) {
	e.resolvedMatchers = map[string]interface{}{}
	for name, _ := range e.AST {
		_, ok := e.resolvedMatchers[name]
		if ok {
			// Matcher already resolved
			continue
		}
		_, err := e.resolveMatcher(e.currentState, name)
		if err != nil {
			return nil, err
		}
	}
	return e.resolvedMatchers, nil
}
