package eachers

import (
	"github.com/onsi/gomega"
	gomegaTypes "github.com/onsi/gomega/types"
)

type eachTo struct {
	matcherFactory func(interface{}) gomegaTypes.GomegaMatcher
	values         []interface{}
	passing        bool

	erroredMatcher gomegaTypes.GomegaMatcher
}

func Each(matcherFactory func(interface{}) gomegaTypes.GomegaMatcher, values ...interface{}) gomegaTypes.GomegaMatcher {
	return &eachTo{
		matcherFactory: matcherFactory,
		values:         values,
		passing:        true,
	}
}

func (e *eachTo) Match(actual interface{}) (bool, error) {
	if !e.passing {
		return false, nil
	}

	for _, v := range e.values {

		success, err := e.testValue(actual, v)

		if err != nil || !success {
			return false, err
		}

		e.passing = e.passing && success
		e.values = e.values[1:]
	}

	return e.passing, nil
}

func (e *eachTo) FailureMessage(actual interface{}) string {
	return e.erroredMatcher.FailureMessage(actual)
}

func (e *eachTo) NegatedFailureMessage(actual interface{}) string {
	return e.erroredMatcher.NegatedFailureMessage(actual)
}

func (e *eachTo) testValue(actual, value interface{}) (bool, error) {
	var rx interface{}
	success, err := gomega.Receive(&rx).Match(actual)
	if !success || err != nil {
		e.erroredMatcher = gomega.Receive(&rx)
		return false, err
	}

	success, err = e.matcherFactory(value).Match(rx)
	if !success || err != nil {
		e.erroredMatcher = e.matcherFactory(value)
	}

	return success, err
}
