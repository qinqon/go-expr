package main

import (
	"fmt"
	"io"
	"strconv"
)

type Parser struct {
	lexer *Lexer
}

func NewParser(reader io.Reader) *Parser {
	return &Parser{
		lexer: NewLexer(reader),
	}
}

func badExpressionError(token *Token, msg string) error {
	return fmt.Errorf("bad expression (col %d, lit %s): %s", token.Position.Column, token.Literal, msg)
}

func (p *Parser) parseArgument() (*Argument, *Token, error) {
	argument := Argument{}
	var currentToken, lastToken *Token
	for {
		var err error
		currentToken, err = p.lexer.Lex()
		if err != nil {
			return nil, nil, err
		}
		if currentToken.Type.IsOperator() || currentToken.Type == PIPE || currentToken.Type == EOF {
			break
		}
		if currentToken.Type == IDENT {
			argument.AppendStepToPath(Step{Identifier: &currentToken.Literal})
		} else if currentToken.Type == INT {
			idx, err := strconv.Atoi(currentToken.Literal)
			if err != nil {
				return nil, nil, err
			}
			argument.AppendStepToPath(Step{Index: &idx})
		} else if currentToken.Type == PATH {
			if lastToken == nil {
				return nil, nil, badExpressionError(currentToken, "cannot start with dot")
			} else if lastToken.Type == PATH {
				return nil, nil, badExpressionError(currentToken, "just one dot can be used")
			} else if lastToken.Type != IDENT && lastToken.Type != INT {
				return nil, nil, badExpressionError(currentToken, "only dot with identifiers or integer can be used on path expression")
			}
		} else if currentToken.Type == STRING {
			if lastToken != nil {
				return nil, nil, badExpressionError(currentToken, "path expressions and strings cannot be mixed")
			}
			argument.String = &currentToken.Literal
		}
		lastToken = currentToken
	}
	return &argument, currentToken, nil
}

func (p *Parser) parseNode() (*Node, *Token, error) {
	node := Node{
		Expression: Expression{
			Arguments: []Argument{},
		},
	}
	// LHS argument
	argument, currentToken, err := p.parseArgument()
	if err != nil {
		return nil, nil, err
	}
	node.Expression.Arguments = append(node.Expression.Arguments, *argument)

	if currentToken.Type.IsOperator() {
		switch currentToken.Type {
		case ASSIGN:
			node.Expression.Operator = Replace
		case EQUAL:
			node.Expression.Operator = Equal
		case MERGE:
			node.Expression.Operator = Merge
		default:
			return nil, nil, badExpressionError(currentToken, "unknown operator")
		}

		// RHS argument
		argument, currentToken, err = p.parseArgument()
		if err != nil {
			return nil, nil, err
		}
		node.Expression.Arguments = append(node.Expression.Arguments, *argument)
	}
	return &node, currentToken, nil
}
func (p *Parser) Parse() (*Node, error) {
	node, currentToken, err := p.parseNode()
	if err != nil {
		return nil, err
	}

	for currentToken.Type == PIPE {
		node, currentToken, err = p.parseNode()
		if err != nil {
			return nil, err
		}
		node.Pipe = node
	}
	return node, nil
}
