package jumpparser

import (
	"testing"

	"github.com/alexasa79/hless/internal/utils"
)

var assert = utils.Assert

func TestTokenizer(t *testing.T) {
	tokens := tokenize("100")
	assert(len(tokens) == 1)
	assert(tokens[0].Type == TOKEN_NUM)
	assert(tokens[0].Value == "100")

	tokens = tokenize(" 100% ")
	assert(len(tokens) == 2)
	assert(tokens[0].Type == TOKEN_NUM)
	assert(tokens[0].Value == "100")
	assert(tokens[1].Type == TOKEN_PERCENT)
	assert(tokens[1].Value == "%")

	tokens = tokenize(" 100% 100")
	assert(len(tokens) == 3)
	assert(tokens[0].Type == TOKEN_NUM)
	assert(tokens[0].Value == "100")
	assert(tokens[1].Type == TOKEN_PERCENT)
	assert(tokens[1].Value == "%")
	assert(tokens[2].Type == TOKEN_NUM)
	assert(tokens[2].Value == "100")

	tokens = tokenize(" + end")
	assert(len(tokens) == 2)
	assert(tokens[0].Type == TOKEN_PLUS)
	assert(tokens[0].Value == "+")
	assert(tokens[1].Type == TOKEN_WORD)
	assert(tokens[1].Value == "end")

	tokens = tokenize("298.392% -")
	assert(len(tokens) == 3)
	assert(tokens[0].Type == TOKEN_NUM)
	assert(tokens[0].Value == "298.392")
	assert(tokens[1].Type == TOKEN_PERCENT)
	assert(tokens[1].Value == "%")
	assert(tokens[2].Type == TOKEN_MINUS)
	assert(tokens[2].Value == "-")
}

func TestParser(t *testing.T) {
	tokens := tokenize("100%")
	j, err := parse(tokens)
	assert(err == nil)
	assert(j.Lines == nil)
	assert(j.Offset != nil && *j.Offset == 100.0)

	tokens = tokenize("100% - 23")
	j, err = parse(tokens)
	assert(err == nil)
	assert(j.Offset != nil && *j.Offset == 100.0)
	assert(j.Lines != nil && *j.Lines == -23)

	tokens = tokenize("50.7%-23+30")
	j, err = parse(tokens)
	assert(err == nil)
	assert(j.Offset != nil && *j.Offset == 50.7)
	assert(j.Lines != nil && *j.Lines == 7)

	tokens = tokenize("-10")
	j, err = parse(tokens)
	assert(err == nil)
	assert(j.Offset == nil)
	assert(j.Lines != nil && *j.Lines == -10)

	tokens = tokenize("end+3")
	j, err = parse(tokens)
	assert(err == nil)
	assert(j.Offset != nil && *j.Offset == 100)
	assert(j.Lines != nil && *j.Lines == 3)

	tokens = tokenize("junk")
	j, err = parse(tokens)
	assert(err != nil)
}
