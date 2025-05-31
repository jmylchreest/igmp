// Copyright © 2016 John Mylchreest <jmylchreest@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package main

import (
	"github.com/jmylchreest/igmpqd/cmd"
	"strconv"
)

// Variables to be set by linker flags
var (
	GitCommit   string
	GitDescribe string
	BuildTime   string // Unix epoch seconds, as a string from ldflags
)

func main() {
	// Assign values from ldflags (main package) to cmd package variables
	cmd.GitCommit = GitCommit
	cmd.GitDescribe = GitDescribe

	if bt, err := strconv.ParseInt(BuildTime, 10, 64); err == nil {
		cmd.BuildTime = bt
	}
	// If BuildTime is not a valid int, cmd.BuildTime will remain 0 (its zero value)
	// cmd/version.go handles the case where cmd.BuildTime is 0.

	cmd.Execute()
}
