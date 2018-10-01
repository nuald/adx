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
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

func newCmd(name string, args ...string) *exec.Cmd {
	fmt.Println(name, strings.Join(args, " "))

	// #nosec
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
	IsCtor      bool        `xml:"-"`
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
	genIntermediate(srcDir string) []byte
	combineIntermediate(a []byte, b []byte) []byte
	genClasses(xmlContent []byte) []Class
}

var compoundRe = regexp.MustCompile("<ref refid=\"(\\w+)\" kindref=\"compound\">(\\w+)</ref>")

func getText(raw string) template.HTML {
	m := compoundRe.FindStringSubmatch(raw)
	if m != nil {
		// #nosec
		return template.HTML(fmt.Sprintf("<a href=\"#%s\">%s</a>", m[1], m[2]))
	}
	// #nosec
	return template.HTML(raw)
}

func getPlainText(raw string) string {
	return compoundRe.ReplaceAllString(raw, "$2")
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
	Description  Raw          `xml:"briefdescription>para"`
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
	Description Raw          `xml:"briefdescription>para"`
}

func genDoxyMethodReturn(member MemberDef, returnDesc template.HTML) Returns {
	returnType := getText(member.Type.RawXML)
	return Returns{
		Type:        returnType,
		Description: returnDesc,
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
		Description: getPlainText(member.Description.RawXML),
		Returns:     ret,
		Parameters:  parameters,
		Access:      access,
	}
}

func genDoxyClass(def CompoundDef) Class {
	cls := Class{
		Name:        def.Name,
		Description: getPlainText(def.Description.RawXML),
		Ref:         def.Ref,
	}
	for _, section := range def.Sections {
		sectionKind := section.Kind
		if sectionKind == "public-static-attrib" {
			for _, member := range section.Members {
				prop := Property{
					Name:        member.Name,
					Description: getPlainText(member.Description.RawXML),
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

func createDir(dir string) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		log.Fatal(err)
	}
}

var generators = map[string]generator{
	"js":   new(js),
	"java": new(java),
}

func renderTemplate(tplFile string, data interface{}) []byte {
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

	return buf.Bytes()
}

func renderHTML(title string, namespaces map[string][]Class) []byte {
	return renderTemplate("data/default.html", struct {
		Title      string
		Namespaces map[string][]Class
	}{
		title,
		namespaces,
	})
}

func printUsage() {
	fmt.Println("Usage: adx [-conf=(yaml-file)] -lang=(lang) [-src=(src-dir)]+ [-xml=(xml-file)]+ -title=(title) -out=(out.[html|pdf|xml])")
	fmt.Println("Produces the code's auto-generated documentation in HTML, PDF or XML.")
	fmt.Println()
	fmt.Println("Flags:")
	flag.PrintDefaults()
}

func save(content []byte, out string) {
	if err := ioutil.WriteFile(out, content, 0644); err != nil {
		log.Fatal(err)
	}
}

func savePdf(html []byte, out string) {
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

	if _, err = tmpfile.Write(html); err != nil {
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
		for i, method := range cls.Methods {
			returnType := method.Returns.Type
			returnDesc := method.Returns.Description
			noReturnInfo := returnType == "" && returnDesc == ""
			cls.Methods[i].Returns.Skip = returnType == "void" || noReturnInfo
		}
		namespaces[ns] = append(namespaces[ns], cls)
	}
	return namespaces
}

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "the string flags"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func getIntermediateContent(srcDirs arrayFlags, gen generator) []byte {
	var intermediateContent []byte
	for _, srcDir := range srcDirs {
		content := gen.genIntermediate(srcDir)
		if intermediateContent != nil {
			intermediateContent = gen.combineIntermediate(intermediateContent, content)
		} else {
			intermediateContent = content
		}
	}
	return intermediateContent
}

// AdxResult XML struct
type AdxResult struct {
	XMLName xml.Name `xml:"adx"`
	Classes []Class  `xml:"classes"`
}

func combineClasses(classes []Class, xmlFiles arrayFlags) []Class {
	var v AdxResult

	for _, xmlFile := range xmlFiles {
		// #nosec
		xmlContent, err := ioutil.ReadFile(xmlFile)
		if err != nil {
			log.Fatal(err)
		}

		if err := xml.Unmarshal(xmlContent, &v); err != nil {
			log.Fatal(err)
		}

		classes = append(classes, v.Classes...)
	}
	return classes
}

func renderXML(classes []Class) []byte {
	v := AdxResult{
		Classes: classes,
	}
	result, err := xml.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	return result
}

func findGenerator(conf string, lang string) (generator, bool) {
	if conf != "" {
		// #nosec
		data, err := ioutil.ReadFile(conf)
		if err != nil {
			log.Fatal(err)
		}
		var config map[string]Language
		err = yaml.Unmarshal(data, &config)
		if err != nil {
			log.Fatal(err)
		}
		langConfig, ok := config[lang]
		if ok {
			return createCustomGen(langConfig), true
		}
		return nil, false
	}
	gen, ok := generators[lang]
	return gen, ok
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

	var srcDirs arrayFlags
	var xmlFiles arrayFlags
	flag.Var(&srcDirs, "src", "the source code dir(s)")
	flag.Var(&xmlFiles, "xml", "the input XML file(s)")

	title := flag.String("title", "", "the document title")
	conf := flag.String("conf", "", "the configuration file for the custom languages")
	out := flag.String("out", "", "the output file (the format is based on its extension)")
	flag.Parse()
	gen, ok := findGenerator(*conf, *lang)
	if !ok {
		fmt.Printf("Can't find a documentation generator for %s\n\n", *lang)
		printUsage()
	} else {
		intermediateContent := getIntermediateContent(srcDirs, gen)
		classes := gen.genClasses(intermediateContent)
		combined := combineClasses(classes, xmlFiles)
		createDir(filepath.Dir(*out))
		ext := filepath.Ext(*out)
		if ext == ".xml" {
			save(renderXML(combined), *out)
		} else {
			html := renderHTML(*title, normalize(combined))
			if ext == ".html" {
				save(html, *out)
			} else if ext == ".pdf" {
				savePdf(html, *out)
			} else {
				fmt.Printf("Can't find a printer for %s format\n\n", ext)
				printUsage()
			}
		}
	}
}
