package main

import (
	"fmt"
)

type MatchersEmitter struct {
	AST              map[string]*Node
	resolvedMatchers map[string]interface{}
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
	}
	return nil, nil
}

func (e MatchersEmitter) resolvePath(path []Step) (interface{}, error) {
	var result interface{}
	// Use the resolved matcher as the input
	if path[0].Identifier != nil && *path[0].Identifier == "matchers" {
		result = e.resolvedMatchers[*path[1].Identifier]
		path = path[2:len(path)]
	}
	//TODO: pass currentState ?
	for _, step := range path {
		if step.Identifier != nil {
			result = result.(map[interface{}]interface{})[*step.Identifier]
		} else if step.Index != nil {
			result = result.([]interface{})[*step.Index]
		}
	}
	return result, nil
}

func (e MatchersEmitter) resolveFilterArgument(argument Argument) (*filterArgument, error) {
	if argument.Path != nil {
		result, err := e.resolvePath(argument.Path)
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

func (e MatchersEmitter) resolveEqual(currentState interface{}, arguments []Argument) (interface{}, error) {
	lhs := arguments[0]
	rhs := arguments[1]
	if lhs.Path == nil {
		return nil, fmt.Errorf("bad filter expression: left argument should be a path")
	}
	argument, err := e.resolveFilterArgument(rhs)
	if err != nil {
		return nil, err
	}
	result, err := filter(currentState, lhs.Path, argument)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (e MatchersEmitter) Emit(currentState interface{}) (map[string]interface{}, error) {
	e.resolvedMatchers = map[string]interface{}{}
	for name, ast := range e.AST {
		var resolvedMatcher interface{}
		switch ast.Expression.Operator {
		case Equal:
			var err error
			resolvedMatcher, err = e.resolveEqual(currentState, ast.Expression.Arguments)
			if err != nil {
				return nil, err
			}
		}
		//TODO Pipe
		e.resolvedMatchers[name] = resolvedMatcher
	}
	return e.resolvedMatchers, nil
}
