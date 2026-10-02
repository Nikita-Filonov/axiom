package testjunit

import "encoding/xml"

type xmlSuites struct {
	XMLName  xml.Name   `xml:"testsuites"`
	Tests    int        `xml:"tests,attr"`
	Failures int        `xml:"failures,attr"`
	Skipped  int        `xml:"skipped,attr"`
	Time     string     `xml:"time,attr"`
	Suites   []xmlSuite `xml:"testsuite"`
}

type xmlSuite struct {
	Name     string    `xml:"name,attr"`
	Tests    int       `xml:"tests,attr"`
	Failures int       `xml:"failures,attr"`
	Skipped  int       `xml:"skipped,attr"`
	Time     string    `xml:"time,attr"`
	Cases    []xmlCase `xml:"testcase"`
}

type xmlCase struct {
	ClassName string      `xml:"classname,attr"`
	Name      string      `xml:"name,attr"`
	Time      string      `xml:"time,attr"`
	Failure   *xmlOutcome `xml:"failure,omitempty"`
	Skipped   *xmlOutcome `xml:"skipped,omitempty"`
}

type xmlOutcome struct {
	Message string `xml:"message,attr"`
	Text    string `xml:",chardata"`
}
