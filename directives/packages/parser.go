package packages

import (
	"errors"
)

type parseResult interface {
	Result() (Package, error)
}

// Parse ...
func Parse(s string) (Package, error) {
	p := &parser{
		lexer: lex(s),
		state: nameFn,
	}

	return p.run()
}

type parserStateFn func(*parser) parserStateFn

type parser struct {
	err    error
	result Package
	state  parserStateFn // state of the parser
	lexer  *lexer
}

func (t *parser) run() (Package, error) {
	for t.state != nil {
		t.state = t.state(t)
	}

	return t.result, t.err
}

func nameFn(p *parser) parserStateFn {
	token := p.lexer.NextToken()
	switch token.typ {
	case tokenFin:
		return nil
	case tokenText:
		p.result.Name = token.val
		return versionFn
	default:
		return errorFn(p, errors.New("unexpected token"))
	}
}

func versionFn(p *parser) parserStateFn {
	token := p.lexer.NextToken()
	switch token.typ {
	case tokenFin:
		return nil
	case tokenText:
		p.result.Version = token.val
		return archFn
	default:
		return errorFn(p, errors.New("unexpected token"))
	}
}

func archFn(p *parser) parserStateFn {
	token := p.lexer.NextToken()
	switch token.typ {
	case tokenFin:
		return nil
	case tokenText:
		p.result.Architecture = token.val
		return repositoryFn
	default:
		return errorFn(p, errors.New("unexpected token"))
	}
}

func repositoryFn(p *parser) parserStateFn {
	token := p.lexer.NextToken()
	switch token.typ {
	case tokenFin:
		return nil
	case tokenText:
		p.result.Repository = token.val
		return nil
	default:
		return errorFn(p, errors.New("unexpected token"))
	}
}

func errorFn(p *parser, err error) parserStateFn {
	p.err = err
	return nil
}
