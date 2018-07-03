package main

import (
	"encoding/xml"
	"log"
)

type js struct{}

func (j js) genClasses(xmlContent []byte) []Class {
	type Result struct {
		XMLName xml.Name `xml:"jsdoc"`
		Classes []Class  `xml:"classes"`
	}

	var v Result
	if err := xml.Unmarshal(xmlContent, &v); err != nil {
		log.Fatal(err)
	}

	return v.Classes
}

func (j js) genIntermediate(srcDir string) []byte {
	cmd := newCmd("jsdoc", "-t", "templates/haruki", srcDir, "-d", "console",
		"-q", "format=xml")
	out, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	return out
}

func (j js) combineIntermediate(a []byte, b []byte) []byte {
	type Result struct {
		XMLName xml.Name `xml:"jsdoc"`
		Classes string   `xml:",innerxml"`
	}
	var aContent Result
	if err := xml.Unmarshal(a, &aContent); err != nil {
		log.Fatal(err)
	}

	var bContent Result
	if err := xml.Unmarshal(b, &bContent); err != nil {
		log.Fatal(err)
	}

	aContent.Classes = aContent.Classes + bContent.Classes
	output, err := xml.Marshal(aContent)
	if err != nil {
		log.Fatal(err)
	}

	return output
}
