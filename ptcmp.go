// Copyright 2009 The ptcmp Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// ptcmp is a command line utility for processing before/after XML
// output of Grant Skinner's PerformanceTest v2 Beta
package main

import (
	"encoding/xml"
	"io"
	"log"
	"os"
	"text/template"
)

//<TestCollection playerVersion="MAC 11,5,500,97" isDebugger="false" playerType="PlugIn" os="Mac OS 10.8.2" manufacturer="Google Macintosh" cpuArchitecture="x86">
//  <TestSuite name="Constructor" time="6883" tareTime="1" initTime="0" description="Comparing constructor. 100000 loops." error="false">
//    <MethodTest name="new BigDecimal(1) x 100000" time="17.8" min="17" max="20" deviation="0.169" description="Create BigDecimal with int parameter 1" memory="79" retainedMemory="4">
// ...
//    <MethodTest name="new BigDecimal(5432) x 100000" time="106.5" min="104" max="110" deviation="0.056" description="Create BigDecimal with int parameter 5432" memory="112" retainedMemory="0">

type Test struct {
	Name        string  `xml:"name,attr"`
	Description string  `xml:"description,attr"`
	Time        float32 `xml:"time,attr"`
	Memory      float32 `xml:"memory,attr"`
}

type TestSuite struct {
	Name        string  `xml:"name,attr"`
	Description string  `xml:"description,attr"`
	Time        float32 `xml:"time,attr"`
	Error       bool    `xml:"error,attr"`
	Tests       []Test  `xml:"MethodTest"`
}

type TestCollection struct {
	Suites []TestSuite `xml:"TestSuite"`
}

func parseTests(file string) TestCollection {
	f, err := os.Open(file)
	if err != nil {
		log.Fatalf("Open failed: %v\n", err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		log.Fatalf("Stat failed: %v\n", err)
	}

	buf := make([]byte, fi.Size())
	n, err := io.ReadFull(f, buf)
	if err != nil {
		log.Fatalf("Read failed %v\n", err)
	} else if int64(n) != fi.Size() {
		log.Fatalf("Read length mismatch, expected %v, got %v\n", fi.Size(), n)
	}

	tc := &TestCollection{}
	err = xml.Unmarshal(buf, tc)
	if err != nil {
		log.Fatalf("Unmarshall failed %v\n", err)
	}
	return *tc
}

func change(before, after float32) float32 {
	if before == after {
		return 0
	}
	change := 100*after/before - 100
	return change
}

var funcMap = template.FuncMap{"change": change}

var html = template.Must(template.New("html").Funcs(funcMap).Parse(`{{$oldSuites := .OldSuites}}{{$testSuites := .NewSuites}}<table>{{range $suiteIndex, $os := $oldSuites}}{{$ns := index $testSuites $suiteIndex}}
	<tr>
		<th colspan="5">{{$os.Description}}</th>
	</tr>
	<tr>
		<th align="left">Test</th>
		<th align="right">Original time</th>
		<th align="right">Original memory</th>
		<th align="right">Time (change %)</th>
		<th align="right">Memory (change %)</th>
	</tr>{{range $testIndex, $bt := $os.Tests}}{{$tt := index $ns.Tests $testIndex}}
	<tr>
		<td><code>{{$bt.Name}}</code></td>
		<td align="right">{{$bt.Time}}</td>
		<td align="right">{{$bt.Memory}}</td>
		<td align="right">{{$tt.Time}} ({{change $bt.Time $tt.Time | printf "%0.1f"}})</td>
		<td align="right">{{$tt.Memory}} ({{change $bt.Memory $tt.Memory | printf "%0.1f"}})</td>
	</tr>{{end}}{{end}}
</table>
`))

var ascii = template.Must(template.New("ascii").Funcs(funcMap).Parse(`{{/*
*/}}{{$oldSuites := .OldSuites}}{{/*
*/}}{{$testSuites := .NewSuites}}{{/*
*/}}{{range $suiteIndex, $os := $oldSuites}}{{/*
*/}}{{$ns := index $testSuites $suiteIndex}}{{/*
*/}}{{$os.Description}}
{{printf "%-*s %8s %8s %8s %8s %8s %8s" 60 "TEST" "OLD MS" "NEW MS" "DELTA" "OLD MEM" "NEW MEM" "DELTA"}}
{{range $testIndex, $bt := $os.Tests}}{{/*
*/}}{{$tt := index $ns.Tests $testIndex}}{{/*
*/}}{{printf "%-*s" 60 $bt.Name}}{{/*
*/}}{{printf "%.2f" $bt.Time | printf " %8s"}}{{/*
*/}}{{printf "%.2f" $tt.Time | printf " %8s"}}{{/*
*/}}{{change $bt.Time $tt.Time | printf "%.2f" | printf " %8s"}}{{/*
*/}}{{printf "%.2f" $bt.Memory | printf " %8s"}}{{/*
*/}}{{printf "%.2f" $tt.Memory | printf " %8s"}}{{/*
*/}}{{change $bt.Memory $tt.Memory | printf "%.2f" | printf " %8s"}}
{{end}}
{{end}}`))

//	{{change $bt.Time $tt.Time | printf "   % 3.1f "}} {{printf "%8.2f %8.2f" $bt.Memory $tt.Memory}}  {{change $bt.Memory $tt.Memory | printf "%3.1f"}}

// {{printf "%8.2f" $bt.Memory}} {{printf "%8.2f" $tt.Time}} ({{change $bt.Time $tt.Time | printf "%3.1f"}}) {{printf "%8.2f" $tt.Memory}} ({{change $bt.Memory $tt.Memory | printf "%3.1f"}})

//{{printf "%-60s" $bt.Name}}{{printf "%8.2f" $bt.Time}} {{printf "%8.2f" $bt.Memory}} {{printf "%8.2f" $tt.Time}} ({{change $bt.Time $tt.Time | printf "%3.1f"}}) {{printf "%8.2f" $tt.Memory}} ({{change $bt.Memory $tt.Memory | printf "%3.1f"}})

func main() {
	log.SetFlags(0)
	log.SetPrefix("")

	args := os.Args[1:]

	templ := ascii
	if len(args) > 0 && args[0] == "-html" {
		templ = html
		args = args[1:]
	}

	if len(args) != 2 {
		log.Fatalf("Usage: %s [-html] <old.xml> <new.xml>\n", os.Args[0])
	}

	o := parseTests(args[0])
	n := parseTests(args[1])

	err := templ.Execute(os.Stdout, map[string]interface{}{
		"OldSuites": o.Suites,
		"NewSuites": n.Suites})
	if err != nil {
		log.Fatalf("Failed to execute template: %v\n", err)
	}
}
