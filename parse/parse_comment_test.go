// go-rst - A reStructuredText parser for Go
// 2014 (c) The go-rst Authors
// MIT Licensed. See LICENSE for details.

// To enable debug output when testing, use "go test -debug"

package parse

import "testing"

func TestParseCommentGood0000(t *testing.T) {
	// A single comment
	testPath := testPathFromName("00.00-comment")
	test := LoadParseTest(t, testPath)
	pTree := parseTest(t, test)
	eNodes := test.expectNodes()
	checkParseNodes(t, eNodes, pTree.Nodes, testPath)
}

func TestParseCommentBlockGood0001(t *testing.T) {
	// A single comment split with a newline
	testPath := testPathFromName("00.01-comment-block")
	test := LoadParseTest(t, testPath)
	pTree := parseTest(t, test)
	eNodes := test.expectNodes()
	checkParseNodes(t, eNodes, pTree.Nodes, testPath)
}

func TestParseCommentBlockOnSecondLineGood0100(t *testing.T) {
	// A comment block that begins on the second line after the comment
	// mark
	testPath := testPathFromName("01.00-comment-block-second-line")
	test := LoadParseTest(t, testPath)
	pTree := parseTest(t, test)
	eNodes := test.expectNodes()
	checkParseNodes(t, eNodes, pTree.Nodes, testPath)
}

func TestParseTwoCommentsGood0200(t *testing.T) {
	// One comment after another
	testPath := testPathFromName("02.00-two-comments")
	test := LoadParseTest(t, testPath)
	pTree := parseTest(t, test)
	eNodes := test.expectNodes()
	checkParseNodes(t, eNodes, pTree.Nodes, testPath)
}
