package main

import (
	"html/template"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var classToken = "Class:"
var methodToken = "Method:"
var constructorToken = "Constructor:"
var staticMethodToken = "Static Method:"
var propertyToken = "Property:"
var staticPropertyToken = "Static Property:"

// Language configuration
type Language struct {
	Extensions []string
	Docstrings struct {
		Type      string
		Format    string
		Parameter string
		Return    string
	}
}

type custom struct {
	language Language
}

func createCustomGen(language Language) generator {
	gen := new(custom)
	gen.language = language
	return gen
}

func (c custom) setConf(conf string) {}

func (c custom) genIntermediate(srcDir string) []byte {
	var content []byte

	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			ext := filepath.Ext(info.Name())
			for _, possibleExt := range c.language.Extensions {
				if possibleExt == ext {
					// #nosec
					file, err := os.ReadFile(path)
					if err != nil {
						return err
					}
					content = append(content, file...)
				}
			}
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
	return content
}

func (c custom) combineIntermediate(a []byte, b []byte) []byte {
	return append(a, b...)
}

func extractBlocks(lines []string, begin string, middle string, end string) [][]string {
	var result [][]string
	blockStarted := false
	var current []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, begin) && !blockStarted {
			blockStarted = true
			right := strings.TrimSpace(trimmed[len(begin):])
			if right != "" {
				current = append(current, right)
			}
			continue
		}

		if strings.HasPrefix(trimmed, end) && blockStarted {
			blockStarted = false
			result = append(result, current)
			current = nil
			continue
		}

		if blockStarted {
			if strings.HasPrefix(trimmed, middle) {
				right := strings.TrimSpace(trimmed[len(middle):])
				if right != "" {
					current = append(current, right)
				}
			} else {
				current = append(current, trimmed)
			}
		}
	}
	return result
}

func extractLines(lines []string, begin string) [][]string {
	var result [][]string
	var current []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, begin) {
			right := strings.TrimSpace(trimmed[len(begin):])
			if right != "" {
				current = append(current, right)
			}
		} else {
			if current != nil {
				result = append(result, current)
			}
			current = nil
		}
	}
	return result
}

func reSubMatchMap(r *regexp.Regexp, str string) map[string]string {
	match := r.FindStringSubmatch(str)
	if match != nil {
		subMatchMap := make(map[string]string)
		for i, name := range r.SubexpNames() {
			if i != 0 {
				subMatchMap[name] = match[i]
			}
		}

		return subMatchMap
	}
	return nil
}

func findMethodTokens(context *string, method *Method, line string,
	paramRe *regexp.Regexp, returnRe *regexp.Regexp) *string {
	if context != nil {
		if *context != methodToken {
			return context
		}
	}

	if method != nil {
		parameter := reSubMatchMap(paramRe, line)
		returnValue := reSubMatchMap(returnRe, line)
		if parameter != nil {
			method.Parameters = append(method.Parameters, Parameter{
				Name:        parameter["name"],
				Description: parameter["description"],
			})
		} else if returnValue != nil {
			// #nosec
			method.Returns = Returns{
				Description: template.HTML(
					strings.TrimSpace(returnValue["description"])),
			}
		} else {
			return context
		}
		// Any token means that's the method description is finished
		return nil
	}
	return context
}

func findClassTokens(cls *Class, line string) bool {
	propToken := "@property"
	ctorToken := "@constructor"

	if strings.HasPrefix(line, propToken) {
		tokens := strings.SplitN(
			strings.TrimSpace(line[len(propToken):]), " ", 2)
		cls.Properties = append(cls.Properties, Property{
			Name:        tokens[0],
			Description: tokens[1],
		})
	} else if strings.HasPrefix(line, ctorToken) {
		var params []Parameter
		for _, prop := range cls.Properties {
			params = append(params, Parameter{
				Name:        prop.Name,
				Description: prop.Description,
			})
		}
		cls.Constructors = append(cls.Constructors, Method{
			Description: strings.TrimSpace(line[len(ctorToken):]),
			Parameters:  params,
		})
	} else {
		return false
	}
	return true
}

func addMethod(cls *Class, method *Method) {
	if method != nil {
		if method.IsCtor {
			cls.Constructors = append(cls.Constructors, *method)
		} else {
			cls.Methods = append(cls.Methods, *method)
		}
	}
}

func addProperty(cls *Class, property *Property) {
	if property != nil {
		cls.Properties = append(cls.Properties, *property)
	}
}

func appendClass(classes []Class, cls *Class, method *Method, property *Property) []Class {
	addMethod(cls, method)
	addProperty(cls, property)
	if cls != nil {
		classes = append(classes, *cls)
	}
	return classes
}

func updateDescriptions(line string, context *string, cls *Class, method *Method, property *Property) {
	if context != nil {
		if *context == classToken {
			if cls.Description == "" {
				cls.Description = line
			} else {
				cls.Description += "\n" + line
			}
		}
		if *context == methodToken {
			if method.Description == "" {
				method.Description = line
			} else {
				method.Description += "\n" + line
			}
		}
		if *context == propertyToken {
			if property.Description == "" {
				property.Description = line
			} else {
				property.Description += "\n" + line
			}
		}
	}
}

func getAccessModifier(isStatic bool) string {
	if isStatic {
		return "static"
	}
	return ""
}

func newMethod(line string, isStaticMethod bool, isCtor bool) Method {
	prefix := methodToken
	if isStaticMethod {
		prefix = staticMethodToken
	} else if isCtor {
		prefix = constructorToken
	}
	return Method{
		Name:   strings.TrimSpace(line[len(prefix):]),
		Access: getAccessModifier(isStaticMethod),
		IsCtor: isCtor,
	}
}

func newProperty(line string, isStaticProperty bool) Property {
	prefix := propertyToken
	if isStaticProperty {
		prefix = staticPropertyToken
	}
	return Property{
		Name:   strings.TrimSpace(line[len(prefix):]),
		Access: getAccessModifier(isStaticProperty),
	}
}

func findMethodDeclaration(line string) *Method {
	isMethod := strings.HasPrefix(line, methodToken)
	isStaticMethod := strings.HasPrefix(line, staticMethodToken)
	isCtor := strings.HasPrefix(line, constructorToken)
	if isMethod || isStaticMethod || isCtor {
		methodVar := newMethod(line, isStaticMethod, isCtor)
		return &methodVar
	}
	return nil
}

func findClasses(blocks [][]string, paramRe *regexp.Regexp, returnRe *regexp.Regexp) []Class {
	var classes []Class
	var cls *Class
	var method *Method
	var property *Property
	for _, block := range blocks {
		var context *string
		for _, line := range block {
			if strings.HasPrefix(line, classToken) {
				classes = appendClass(classes, cls, method, property)
				method = nil
				property = nil
				cls = &Class{
					Name: strings.TrimSpace(line[len(classToken):]),
				}
				context = &classToken
				continue
			}
			if cls != nil {
				if newMethodVar := findMethodDeclaration(line); newMethodVar != nil {
					addMethod(cls, method)
					method = newMethodVar
					context = &methodToken
					continue
				}
				isProperty := strings.HasPrefix(line, propertyToken)
				isStaticProperty := strings.HasPrefix(line, staticPropertyToken)
				if isProperty || isStaticProperty {
					addProperty(cls, property)
					newPropertyVar := newProperty(line, isStaticProperty)
					property = &newPropertyVar
					context = &propertyToken
					continue
				}
				if findClassTokens(cls, line) {
					context = nil
				}
			}
			context = findMethodTokens(context, method, line, paramRe, returnRe)
			updateDescriptions(line, context, cls, method, property)
		}
	}
	classes = appendClass(classes, cls, method, property)
	return classes
}

func (c custom) genClasses(content []byte) []Class {
	var blocks [][]string
	lines := strings.Split(string(content), "\n")
	format := strings.Fields(c.language.Docstrings.Format)
	begin := format[0]
	switch c.language.Docstrings.Type {
	case "block":
		if len(format) < 2 {
			log.Fatal("Block docstrings should have a format as the begin and end tokens separated by space.")
		}
		var middle string
		var end string
		if len(format) > 2 {
			middle = format[1]
			end = format[2]
		} else {
			end = format[1]
		}
		blocks = extractBlocks(lines, begin, middle, end)
	case "line":
		if len(format) != 1 {
			log.Fatal("Line docstrings should have a format as the single begin token.")
		}
		blocks = extractLines(lines, begin)
	}
	paramRe := regexp.MustCompile(c.language.Docstrings.Parameter)
	returnRe := regexp.MustCompile(c.language.Docstrings.Return)
	return findClasses(blocks, paramRe, returnRe)
}
