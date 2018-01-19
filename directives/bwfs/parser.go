package bwfs

import (
	"strconv"

	"github.com/pkg/errors"
)

func parse(l *lexer, a *Archive) (err error) {
	var (
		next pState = srcState{}
	)

	for next != nil && err == nil {
		next, err = next.Advance(l.NextToken(), a)
	}

	return err
}

type pState interface {
	Advance(tok token, a *Archive) (pState, error)
}

type doneState struct{}

func (t doneState) Advance(tok token, a *Archive) (pState, error) {
	switch tok.typ {
	case tokenFin:
		return nil, nil
	default:
		return nil, errors.Errorf("expected to be done instead received: %s", tok.val)
	}
}

type srcState struct{}

func (t srcState) Advance(tok token, a *Archive) (next pState, err error) {
	if tok.typ != tokenText {
		return nil, errors.Errorf("expected a filepath token, received: %s", tok.typ)
	}

	a.URI = tok.val

	return pathState{}, nil
}

// type uriState struct{}
//
// func (t uriState) Advance(tok token, a *Archive) (next pState, err error) {
// 	var (
// 		uri *url.URL
// 	)
//
// 	if tok.typ != tokenText {
// 		return nil, errors.Errorf("expected a uri token, received: %s", tok.typ)
// 	}
//
// 	if uri, err = url.Parse(tok.val); err != nil {
// 		return nil, errors.WithStack(err)
// 	}
//
// 	a.URI = uri.String()
//
// 	return pathState{}, nil
// }

type pathState struct{}

func (t pathState) Advance(tok token, a *Archive) (next pState, err error) {
	if tok.typ != tokenText {
		return nil, errors.Errorf("expected a filepath token, received: %s", tok.typ)
	}

	a.Path = tok.val

	return filemodeState{}, nil
}

type filemodeState struct{}

func (t filemodeState) Advance(tok token, a *Archive) (next pState, err error) {
	var (
		mode uint64
	)
	next = userState{}

	if tok.typ != tokenText {
		return nil, errors.Errorf("expected a filepath token, received: %s", tok.typ)
	}

	if isIgnore(tok.val) {
		return next, nil
	}

	if mode, err = strconv.ParseUint(tok.val, 8, 32); err != nil {
		return nil, errors.WithStack(err)
	}

	a.Mode = uint32(mode)

	return next, nil
}

type userState struct{}

func (t userState) Advance(tok token, a *Archive) (next pState, err error) {
	next = groupState{}
	if tok.typ != tokenText {
		return nil, errors.Errorf("expected a filepath token, received: %s", tok.typ)
	}

	if isIgnore(tok.val) {
		return next, nil
	}

	a.Owner = tok.val

	return next, nil
}

type groupState struct{}

func (t groupState) Advance(tok token, a *Archive) (next pState, err error) {
	next = doneState{}
	if tok.typ != tokenText {
		return nil, errors.Errorf("expected a filepath token, received: %s", tok.typ)
	}

	if isIgnore(tok.val) {
		return next, nil
	}

	a.Group = tok.val

	return next, nil
}

func isIgnore(s string) bool {
	return s == CharsetDefault
}
