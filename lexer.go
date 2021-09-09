package main

import (
	"bufio"
	"io"
	"unicode"
)

type TokenType int

const (
	EOF = iota
	ILLEGAL
	IDENT
	INT
	STRING

	PATH // .
	PIPE // |

	begin_operator_section
	ASSIGN // =
	EQUAL  // ==
	MERGE  // +
	end_operator_section
)

var tokens = []string{
	EOF:     "EOF",
	ILLEGAL: "ILLEGAL",
	IDENT:   "IDENT",
	INT:     "INT",
	STRING:  "STRING",

	PATH: ".",
	PIPE: "|",

	ASSIGN: "=",
	EQUAL:  "==",
	MERGE:  "+",
}

func (t TokenType) String() string {
	return tokens[t]
}

func (t TokenType) IsOperator() bool {
	return t > begin_operator_section && t < end_operator_section
}

type Token struct {
	Position Position
	Type     TokenType
	Literal  string
}

type Position struct {
	Column int
}

type Lexer struct {
	pos    Position
	reader *bufio.Reader
}

func NewLexer(reader io.Reader) *Lexer {
	return &Lexer{
		pos:    Position{Column: 0},
		reader: bufio.NewReader(reader),
	}
}

// Lex scans the input for the next token. It returns the position of the token,
// the token's type, and the literal value.
func (l *Lexer) Lex() (*Token, error) {
	// keep looping until we return a token
	for {
		r, _, err := l.reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				return &Token{l.pos, EOF, ""}, nil
			}
			return nil, err
		}

		// update the column to the position of the newly read in rune
		l.pos.Column++

		switch r {
		case '.':
			return &Token{l.pos, PATH, tokens[PATH]}, nil
		case '|':
			return &Token{l.pos, PIPE, tokens[PIPE]}, nil
		case '=':
			if l.lexEqual() {
				return &Token{l.pos, EQUAL, tokens[EQUAL]}, nil
			}
			return &Token{l.pos, ASSIGN, tokens[ASSIGN]}, nil
		case '"':
			startPos := l.pos
			lit := l.lexString()
			return &Token{startPos, STRING, lit}, nil
		default:
			if unicode.IsSpace(r) {
				continue // nothing to do here, just move on
			} else if unicode.IsDigit(r) {
				// backup and let lexInt rescan the beginning of the int
				startPos := l.pos
				l.backup()
				lit := l.lexInt()
				return &Token{startPos, INT, lit}, nil
			} else if unicode.IsLetter(r) || r == '-' {
				// backup and let lexIdent rescan the beginning of the ident
				startPos := l.pos
				l.backup()
				lit := l.lexIdent()
				return &Token{startPos, IDENT, lit}, nil
			} else {
				return &Token{l.pos, ILLEGAL, string(r)}, nil
			}
		}
	}
}

func (l *Lexer) resetPosition() {
	l.pos.Column = 0
}

func (l *Lexer) backup() {
	if err := l.reader.UnreadRune(); err != nil {
		panic(err)
	}

	l.pos.Column--
}

func (l *Lexer) lexString() string {
	lit := ""
	for {
		r, _, err := l.reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				// at the end of the int
				return ""
			}
		}

		l.pos.Column++
		if r != '"' {
			lit = lit + string(r)
		} else {
			return lit
		}
	}
}

func (l *Lexer) lexEqual() bool {
	for {
		r, _, err := l.reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				// at the end of the int
				return false
			}
		}

		l.pos.Column++
		if r == '=' {
			return true
		} else {
			l.backup()
			return false
		}
	}
}

// lexInt scans the input until the end of an integer and then returns the
// literal.
func (l *Lexer) lexInt() string {
	var lit string
	for {
		r, _, err := l.reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				// at the end of the int
				return lit
			}
		}

		l.pos.Column++
		if unicode.IsDigit(r) {
			lit = lit + string(r)
		} else {
			// scanned something not in the integer
			l.backup()
			return lit
		}
	}
}

// lexIdent scans the input until the end of an identifier and then returns the
// literal.
func (l *Lexer) lexIdent() string {
	var lit string
	for {
		r, _, err := l.reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				// at the end of the identifier
				return lit
			}
		}

		l.pos.Column++
		if unicode.IsLetter(r) || r == '-' {
			lit = lit + string(r)
		} else {
			// scanned something not in the identifier
			l.backup()
			return lit
		}
	}
}
