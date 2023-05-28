package main

import (
	"testing"

	. "gopkg.in/check.v1"
)

// Create a test suite
type MinServerIndexSuite struct{}

var _ = Suite(&MinServerIndexSuite{})

// Test case for the minServerIndex function
func (s *MinServerIndexSuite) TestMinServerIndex(c *C) {
	// Set up the test data
	serversPool[0].ConnCnt = 10
	serversPool[1].ConnCnt = 5
	serversPool[2].ConnCnt = 7

	// Call the minServerIndex function
	minIndex := minServerIndex()

	// Check the result
	c.Assert(minIndex, Equals, 1)
}

// Run all the test suites
func Test(t *testing.T) { TestingT(t) }
