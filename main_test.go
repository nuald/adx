package main

import (
	"io/ioutil"
	"testing"
)

func TestKotlin(t *testing.T) {
	gen, ok := findGenerator("fixtures/config.yaml", "kotlin")
	if !ok {
		t.Fatal("Couldn't find kotlin configuration")
	}
	intermediateContent := getIntermediateContent([]string{"fixtures/"}, gen)
	classes := gen.genClasses(intermediateContent)
	xml := string(renderXML(classes))
	data, err := ioutil.ReadFile("fixtures/Foo.xml")
	if err != nil {
		t.Fatal(err)
	}
	str := string(data)
	if xml != str {
		t.Fatalf("XML output doesn't match. Expected:\n%s\nGot:\n%s\n", xml, str)
	}
}

func TestCombineXML(t *testing.T) {
	gen, ok := findGenerator("fixtures/config.yaml", "cpp")
	if !ok {
		t.Fatal("Couldn't find cpp configuration")
	}
	intermediateContent := getIntermediateContent([]string{"fixtures/"}, gen)
	classes := gen.genClasses(intermediateContent)
	xmlFiles := arrayFlags{"fixtures/Foo.xml"}
	combined := combineClasses(classes, xmlFiles)
	xml := string(renderXML(combined))
	data, err := ioutil.ReadFile("fixtures/Combined.xml")
	if err != nil {
		t.Fatal(err)
	}
	str := string(data)
	if xml != str {
		t.Fatalf("XML output doesn't match. Expected:\n%s\nGot:\n%s\n", xml, str)
	}
}
