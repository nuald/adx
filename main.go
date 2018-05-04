package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
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
	Type        template.HTML `xml:"type"`
	Description template.HTML `xml:"description"`
	Skip        bool
}

// Parameter of method
type Parameter struct {
	Name        string        `xml:"name"`
	Type        template.HTML `xml:"type"`
	Description string        `xml:"description"`
	Default     string        `xml:"default"`
	Optional    string        `xml:"optional"`
	Nullable    string        `xml:"nullable"`
}

// Property of class
type Property struct {
	Name        string        `xml:"name"`
	Description string        `xml:"description"`
	Access      string        `xml:"access"`
	Virtual     string        `xml:"virtual"`
	Type        template.HTML `xml:"type"`
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
	Ref          string
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

func getText(raw string) template.HTML {
	r := regexp.MustCompile("<ref refid=\"(\\w+)\" kindref=\"compound\">(\\w+)</ref>")
	m := r.FindStringSubmatch(raw)
	if m != nil {
		// #nosec
		return template.HTML(fmt.Sprintf("<a href=\"#%s\">%s</a>", m[1], m[2]))
	}
	// #nosec
	return template.HTML(raw)
}

// Raw XML info
type Raw struct {
	RawXML string `xml:",innerxml"`
}

// ParameterName info
type ParameterName struct {
	Name string `xml:"parametername"`
}

// ParameterItem info
type ParameterItem struct {
	Names       []ParameterName `xml:"parameternamelist"`
	Description string          `xml:"parameterdescription>para"`
}

// DoxyParameter info
type DoxyParameter struct {
	Kind           string          `xml:"kind,attr"`
	ParameterItems []ParameterItem `xml:"parameteritem"`
}

// SimpleSect info
type SimpleSect struct {
	Kind   string `xml:"kind,attr"`
	RawXML string `xml:",innerxml"`
}

// Paragraph info
type Paragraph struct {
	Parameters     []DoxyParameter `xml:"parameterlist"`
	SimpleSections []SimpleSect    `xml:"simplesect"`
}

// DetailedDesc info
type DetailedDesc struct {
	Paragraphs []Paragraph `xml:"para"`
}

// Param info
type Param struct {
	Type Raw    `xml:"type"`
	Name string `xml:"declname"`
}

// MemberDef info
type MemberDef struct {
	Kind         string       `xml:"kind,attr"`
	Name         string       `xml:"name"`
	Type         Raw          `xml:"type"`
	Description  string       `xml:"briefdescription>para"`
	Parameters   []Param      `xml:"param"`
	DetailedDesc DetailedDesc `xml:"detaileddescription"`
}

// SectionDef info
type SectionDef struct {
	Kind    string      `xml:"kind,attr"`
	Members []MemberDef `xml:"memberdef"`
}

// CompoundDef info
type CompoundDef struct {
	Kind        string       `xml:"kind,attr"`
	Ref         string       `xml:"id,attr"`
	Name        string       `xml:"compoundname"`
	Sections    []SectionDef `xml:"sectiondef"`
	Description string       `xml:"briefdescription>para"`
}

func genDoxyMethodReturn(member MemberDef, returnDesc template.HTML) Returns {
	returnType := getText(member.Type.RawXML)
	noReturnInfo := returnType == "" && returnDesc == ""
	return Returns{
		Type:        returnType,
		Description: returnDesc,
		Skip:        returnType == "void" || noReturnInfo,
	}
}

func genDoxyMethod(member MemberDef, sectionKind string) Method {
	var returnDesc template.HTML
	var parameters []Parameter
	paramDesc := map[string]string{}
	for _, para := range member.DetailedDesc.Paragraphs {
		for _, sect := range para.SimpleSections {
			if sect.Kind == "return" {
				returnDesc = getText(sect.RawXML)
			}
		}
		for _, param := range para.Parameters {
			if param.Kind == "param" {
				for _, item := range param.ParameterItems {
					for _, name := range item.Names {
						paramDesc[name.Name] = item.Description
					}
				}
			}
		}
	}
	for _, param := range member.Parameters {
		name := param.Name
		parameters = append(parameters, Parameter{
			Name:        name,
			Type:        getText(param.Type.RawXML),
			Description: paramDesc[name],
		})
	}
	ret := genDoxyMethodReturn(member, returnDesc)
	access := ""
	if sectionKind == "public-static-func" {
		access = "static"
	}
	return Method{
		Name:        member.Name,
		Description: member.Description,
		Returns:     ret,
		Parameters:  parameters,
		Access:      access,
	}
}

func genDoxyClass(def CompoundDef) Class {
	cls := Class{
		Name:        def.Name,
		Description: def.Description,
		Ref:         def.Ref,
	}
	for _, section := range def.Sections {
		sectionKind := section.Kind
		if sectionKind == "public-static-attrib" {
			for _, member := range section.Members {
				prop := Property{
					Name:        member.Name,
					Description: member.Description,
					Type:        getText(member.Type.RawXML),
					Access:      "static",
				}
				cls.Properties = append(cls.Properties, prop)
			}
		}
		if sectionKind == "public-static-func" || sectionKind == "public-func" {
			for _, member := range section.Members {
				method := genDoxyMethod(member, sectionKind)
				if method.Returns.Type == "" {
					cls.Constructors = append(cls.Constructors, method)
				} else {
					cls.Methods = append(cls.Methods, method)
				}
			}
		}
	}
	return cls
}

func (j java) gen(srcDir string) []Class {
	type Result struct {
		XMLName xml.Name      `xml:"doxygen"`
		Defs    []CompoundDef `xml:"compounddef"`
	}

	javaDoxyfile := renderTemplate("data/java.doxyfile", struct {
		Src string
	}{
		srcDir,
	})

	docsDir := "build/docs"
	if err := os.MkdirAll(docsDir, 0700); err != nil {
		log.Fatal(err)
	}

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
		if _, err = io.WriteString(stdin, javaDoxyfile); err != nil {
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

	var v Result
	if err = xml.Unmarshal(out, &v); err != nil {
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

var generators = map[string]generator{
	"js":   new(js),
	"java": new(java),
}

func renderTemplate(tplFile string, data interface{}) string {
	tpl, err := Asset(tplFile)
	if err != nil {
		log.Fatal(err)
	}

	t, err := template.New("webpage").Parse(string(tpl))
	if err != nil {
		log.Fatal(err)
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, data)
	if err != nil {
		log.Fatal(err)
	}

	return buf.String()
}

func renderHTML(title string, namespaces map[string][]Class) string {
	return renderTemplate("data/default.html", struct {
		Title      string
		Namespaces map[string][]Class
	}{
		title,
		namespaces,
	})
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

func normalize(classes []Class) map[string][]Class {
	namespaces := map[string][]Class{}
	for _, cls := range classes {
		subs := strings.Split(cls.Name, "::")
		ns := "Global"
		if len(subs) > 1 {
			ns = strings.Join(subs[:len(subs)-1], ".")
			cls.Name = subs[len(subs)-1]
		}
		if cls.Ref == "" {
			cls.Ref = ns + cls.Name
		}
		namespaces[ns] = append(namespaces[ns], cls)
	}
	return namespaces
}

func main() {
	// Pre-validation
	_ = MustAsset("data/default.html")
	_ = AssetNames()

	var keys []string
	for key := range generators {
		keys = append(keys, key)
	}
	langDesc := fmt.Sprintf("the source code programming language (%s)",
		strings.Join(keys, ", "))
	lang := flag.String("lang", "", langDesc)
	srcDir := flag.String("src", ".", "the source code dir")
	title := flag.String("title", "", "the document title")
	out := flag.String("out", "", "the output file (the format is based on its extension)")
	flag.Parse()
	gen, ok := generators[*lang]
	if !ok {
		fmt.Printf("Can't find a documentation generator for %s\n\n", *lang)
		printUsage()
	} else {
		html := renderHTML(*title, normalize(gen.gen(*srcDir)))
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
