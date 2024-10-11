package jumpparser

import (
	"fmt"
	"strconv"

	"github.com/alexasa79/hless/internal/utils"
)

// Token types
const (
	TOKEN_NUM = iota
	TOKEN_WORD
	TOKEN_PLUS
	TOKEN_MINUS
	TOKEN_PERCENT
)

type lexer struct {
	input        string
	position     int
	readPosition int
	ch           byte
}

// token represents a lexical token.
type token struct {
	Type  int
	Value string
}

type RelativeJump int

const (
	JUMP_ABSOLUTE = RelativeJump(iota)
	JUMP_FORWARD
	JUMP_BACKWARD
)

type JumpLocation struct {
	Offset *float64
	Lines  *int64
	How    RelativeJump
}

var chk = utils.Check

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9' || ch == '.'
}

func newLexer(input string) *lexer {
	return &lexer{input: input}
}

func (l *lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition += 1
}

func (l *lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' {
		l.readChar()
	}
}

func (l *lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) || l.ch == '.' {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *lexer) readWord() string {
	position := l.position
	for (l.ch >= 'a' && l.ch <= 'z') ||
		(l.ch >= 'A' && l.ch <= 'Z') ||
		(l.ch == '_') ||
		isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *lexer) tokenize() []token {
	var tokens []token
	l.readChar()
	for l.ch != 0 {
		l.skipWhitespace()
		if l.ch == 0 {
			break
		}

		if isDigit(l.ch) {
			val := l.readNumber()
			tokens = append(tokens, token{Type: TOKEN_NUM, Value: val})
		} else {
			if l.ch == '+' {
				tokens = append(tokens, token{Type: TOKEN_PLUS, Value: "+"})
			} else if l.ch == '-' {
				tokens = append(tokens, token{Type: TOKEN_MINUS, Value: "-"})
			} else if l.ch == '%' {
				tokens = append(tokens, token{Type: TOKEN_PERCENT, Value: "%"})
			} else {
				val := l.readWord()
				tokens = append(tokens, token{Type: TOKEN_WORD, Value: val})
			}
			l.readChar()
		}

	}
	return tokens
}

// tokenize takes an input string and returns a slice of tokens.
func tokenize(input string) []token {
	lexer := newLexer(input)
	return lexer.tokenize()
}

// Returns an offset in the file, a number of lines to move forward or backwards from the offset
// and an error.
func parse(tokens []token) (JumpLocation, error) {
	if len(tokens) == 0 {
		return JumpLocation{}, fmt.Errorf("failed to parse")
	}

	res := JumpLocation{}
	add := false
	sub := false
	res.How = JUMP_ABSOLUTE

	for i, t := range tokens {
		switch t.Type {
		case TOKEN_PLUS:
			add = true
		case TOKEN_MINUS:
			sub = true
		case TOKEN_WORD:
			if t.Value != "end" {
				return JumpLocation{}, fmt.Errorf("unexpected token \"%s\"", t.Value)
			}

			if res.Offset == nil {
				res.Offset = new(float64)
			}

			*res.Offset = 100.0

			add = false
			sub = false
		case TOKEN_NUM:
			if i < len(tokens)-1 && tokens[i+1].Type == TOKEN_PERCENT {
				if res.Offset == nil {
					res.Offset = new(float64)
				}

				n, err := strconv.ParseFloat(t.Value, 64)
				chk(err)
				if add {
					res.How = JUMP_FORWARD
				} else if sub {
					res.How = JUMP_BACKWARD
					*res.Offset = n
				}
				*res.Offset = n
			} else {
				if res.Lines == nil {
					res.Lines = new(int64)
				}

				n, err := strconv.Atoi(t.Value)
				chk(err)
				if add {
					res.How = JUMP_FORWARD
				} else if sub {
					res.How = JUMP_BACKWARD
				}
				*res.Lines = int64(n)
			}

			add = false
			sub = false
		}
	}

	return res, nil
}

func Parse(what string) (JumpLocation, error) {
	tokens := tokenize(what)
	return parse(tokens)
}
