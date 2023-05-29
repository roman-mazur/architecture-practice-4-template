package main

import (
	"testing"

	. "gopkg.in/check.v1"
)

type TestBalancer struct{}

var _ = Suite(&TestBalancer{})

func Test(t *testing.T) { TestingT(t) }
