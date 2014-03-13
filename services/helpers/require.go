package helpers

import (
	. "github.com/onsi/gomega"
)

type RequireExpectation struct {
	actual interface{}
}

func Require(actual interface{}) RequireExpectation {
	return RequireExpectation{
		actual: actual,
	}
}

func (e RequireExpectation) To(matcher OmegaMatcher, optionalDescription ...interface{}) {
	result := ExpectWithOffset(1, e.actual).To(matcher, optionalDescription...)
	if !result {
		panic("Failed Required Expectation")
	}
}

func Recover() {
	if r := recover(); r != nil {
		if r != "Failed Required Expectation" {
			panic(r)
		}
	}
}
