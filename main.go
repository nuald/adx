package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func newCmd(name string, args ...string) *exec.Cmd {
	fmt.Println(name, strings.Join(args, " "))

	/* #nosec */
	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	return cmd
}

// Returns info for method
type Returns struct {
	Type        string `xml:"type"`
	Description string `xml:"description"`
}

// Parameter of method
type Parameter struct {
	Name        string `xml:"name"`
	Type        string `xml:"type"`
	Description string `xml:"description"`
	Default     string `xml:"default"`
	Optional    string `xml:"optional"`
	Nullable    string `xml:"nullable"`
}

// Property of class
type Property struct {
	Name        string `xml:"name"`
	Description string `xml:"description"`
	Access      string `xml:"access"`
	Virtual     string `xml:"virtual"`
	Type        string `xml:"type"`
}

// Method of class
type Method struct {
	Name        string      `xml:"name"`
	Description string      `xml:"description"`
	Access      string      `xml:"access"`
	Virtual     string      `xml:"virtual"`
	Parameters  []Parameter `xml:"parameters"`
	Returns     Returns     `xml:"returns"`
}

// Class info
type Class struct {
	Name         string     `xml:"name"`
	Description  string     `xml:"description"`
	Access       string     `xml:"access"`
	Virtual      string     `xml:"virtual"`
	Fires        string     `xml:"fires"`
	Constructors []Method   `xml:"constructor"`
	Methods      []Method   `xml:"functions"`
	Properties   []Property `xml:"properties"`
}

type generator interface {
	gen(srcDir string) []Class
}

type js struct{}

func (j js) gen(srcDir string) []Class {
	type Result struct {
		XMLName xml.Name `xml:"jsdoc"`
		Classes []Class  `xml:"classes"`
	}

	cmd := newCmd("jsdoc", "-t", "templates/haruki", srcDir, "-d", "console",
		"-q", "format=xml")
	out, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}

	var v Result
	if err = xml.Unmarshal(out, &v); err != nil {
		log.Fatal(err)
	}

	return v.Classes
}

type java struct{}

func (j java) gen(srcDir string) []Class {
	return nil
}

var generators = map[string]generator{
	"js":   new(js),
	"java": new(java),
}

func renderTemplate(tplFile string, title string, classes []Class) string {
	tpl, err := ioutil.ReadFile(tplFile)
	if err != nil {
		log.Fatal(err)
	}

	t, err := template.New("webpage").Parse(string(tpl))
	if err != nil {
		log.Fatal(err)
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, struct {
		Title   string
		Classes []Class
	}{
		title,
		classes,
	})
	if err != nil {
		log.Fatal(err)
	}

	return buf.String()
}

func printUsage() {
	fmt.Println("Usage: adx -lang=(lang) -src=(src-dir) -title=(title) -out=(out.[html|pdf])")
	fmt.Println("Converts the source code's auto-generated documentation to HTML and PDF.")
	fmt.Println()
	fmt.Println("Flags:")
	flag.PrintDefaults()
}

func saveHTML(html string, out string) {
	if err := ioutil.WriteFile(out, []byte(html), 0644); err != nil {
		log.Fatal(err)
	}
}

func savePdf(html string, out string) {
	path, err := exec.LookPath("chrome")
	if err != nil {
		path, err = exec.LookPath("/Applications/Google Chrome.app/Contents/MacOS/Google Chrome")
		if err != nil {
			log.Fatal("Can't find Chrome")
		}
	}

	tmpfile, err := ioutil.TempFile("", "tmp")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = os.Remove(tmpfile.Name()); err != nil {
			log.Fatal(err)
		}
	}()

	if _, err = tmpfile.Write([]byte(html)); err != nil {
		log.Fatal(err)
	}

	if err = tmpfile.Close(); err != nil {
		log.Fatal(err)
	}

	cmd := newCmd(path, "--headless", "--disable-gpu", "--print-to-pdf",
		"file://"+tmpfile.Name())
	if err = cmd.Run(); err != nil {
		log.Fatal(err)
	}

	const outpdf = "output.pdf"
	if out != outpdf {
		if err = os.Rename(outpdf, out); err != nil {
			log.Fatal(err)
		}
	}
}

func main() {
	var keys []string
	for key := range generators {
		keys = append(keys, key)
	}
	langDesc := fmt.Sprintf("the source code programming language (%s)",
		strings.Join(keys, ", "))
	lang := flag.String("lang", "", langDesc)
	srcDir := flag.String("src", ".", "the source code dir")
	tplFile := flag.String("template", "default.html", "the HTML template")
	title := flag.String("title", "", "the document title")
	out := flag.String("out", "", "the output file (the format is based on its extension)")
	flag.Parse()
	gen, ok := generators[*lang]
	if !ok {
		fmt.Printf("Can't find a documentation generator for %s\n\n", *lang)
		printUsage()
	} else {
		html := renderTemplate(*tplFile, *title, gen.gen(*srcDir))
		if *out == "" {
			fmt.Println(html)
		} else {
			ext := filepath.Ext(*out)
			if ext == ".html" {
				saveHTML(html, *out)
			} else if ext == ".pdf" {
				savePdf(html, *out)
			} else {
				fmt.Printf("Can't find a printer for %s format\n\n", ext)
				printUsage()
			}
		}
	}
}
