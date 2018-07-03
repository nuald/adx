package main

import (
	"encoding/xml"
	"log"
	"path"
)

type java struct{}

func (j java) genClasses(xmlContent []byte) []Class {
	type Result struct {
		XMLName xml.Name      `xml:"doxygen"`
		Defs    []CompoundDef `xml:"compounddef"`
	}

	var v Result
	if err := xml.Unmarshal(xmlContent, &v); err != nil {
		log.Fatal(err)
	}

	var classes []Class
	for _, def := range v.Defs {
		if def.Kind == "class" {
			classes = append(classes, genDoxyClass(def))
		}
	}

	return classes
}

func (j java) genIntermediate(srcDir string) []byte {
	javaDoxyfile := renderTemplate("data/java.doxyfile", struct {
		Src string
	}{
		srcDir,
	})

	docsDir := "build/docs"
	createDir(docsDir)

	cmd := newCmd("doxygen", "-")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		defer func() {
			if err = stdin.Close(); err != nil {
				log.Fatal(err)
			}
		}()
		if _, err = stdin.Write(javaDoxyfile); err != nil {
			log.Fatal(err)
		}
	}()
	if err = cmd.Run(); err != nil {
		log.Fatal(err)
	}

	xmlDir := path.Join(docsDir, "xml")
	cmd = newCmd("xsltproc", "combine.xslt", "index.xml")
	cmd.Dir = xmlDir
	out, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}

	return out
}

func (j java) combineIntermediate(a []byte, b []byte) []byte {
	type Result struct {
		XMLName xml.Name `xml:"doxygen"`
		Defs    string   `xml:",innerxml"`
	}

	var aContent Result
	if err := xml.Unmarshal(a, &aContent); err != nil {
		log.Fatal(err)
	}

	var bContent Result
	if err := xml.Unmarshal(b, &bContent); err != nil {
		log.Fatal(err)
	}

	aContent.Defs = aContent.Defs + bContent.Defs
	output, err := xml.Marshal(aContent)
	if err != nil {
		log.Fatal(err)
	}

	return output
}
