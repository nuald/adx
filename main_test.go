package main

import (
	"os"
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
	data, err := os.ReadFile("fixtures/Foo.xml")
	if err != nil {
		t.Fatal(err)
	}
	str := string(data)
	if xml != str {
		t.Fatalf("XML output doesn't match. Expected:\n%s\nGot:\n%s\n", xml, str)
	}
}

func TestSwift(t *testing.T) {
	gen, ok := findGenerator("fixtures/config.yaml", "swift")
	if !ok {
		t.Fatal("Couldn't find swift configuration")
	}
	intermediateContent := getIntermediateContent([]string{"fixtures/"}, gen)
	classes := gen.genClasses(intermediateContent)
	xml := string(renderXML(classes))
	data, err := os.ReadFile("fixtures/Bar.xml")
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
	data, err := os.ReadFile("fixtures/Combined.xml")
	if err != nil {
		t.Fatal(err)
	}
	str := string(data)
	if xml != str {
		t.Fatalf("XML output doesn't match. Expected:\n%s\nGot:\n%s\n", xml, str)
	}
}

func TestEmptyIntermediate(t *testing.T) {
	gen, ok := findGenerator("fixtures/config.yaml", "kotlin")
	if !ok {
		t.Fatal("Couldn't find kotlin configuration")
	}
	gen.genClasses(nil)
	generators["js"].genClasses(nil)
	generators["java"].genClasses(nil)
}
