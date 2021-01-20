package main

import (
	"encoding/xml"
	"log"
)

type js struct {
	conf string
}

func (j *js) setConf(conf string) {
	j.conf = conf
}

func (j js) genClasses(xmlContent []byte) []Class {
	type Result struct {
		XMLName xml.Name `xml:"jsdoc"`
		Classes []Class  `xml:"classes"`
	}

	if xmlContent != nil {
		var v Result
		if err := xml.Unmarshal(xmlContent, &v); err != nil {
			log.Fatal(err)
		}
		return v.Classes
	}

	return nil
}

func (j js) genIntermediate(srcDir string) []byte {
	args := []string{"-t", "templates/haruki", srcDir, "-d", "console",
		"-q", "format=xml"}
	if j.conf != "" {
		args = append(args, "-c", j.conf)
	}
	cmd := newCmd("jsdoc", args...)
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
