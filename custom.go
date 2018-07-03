package main

import (
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var classToken = "Class:"
var methodToken = "Method:"

// Language configuration
type Language struct {
	Extensions []string
	Docstrings struct {
		Type   string
		Format string
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
					file, err := ioutil.ReadFile(path)
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

func findMethodTokens(method *Method, line string) bool {
	if method != nil {
		paramToken := "@param"
		returnToken := "@return"

		if strings.HasPrefix(line, paramToken) {
			tokens := strings.SplitN(
				strings.TrimSpace(line[len(paramToken):]), " ", 2)
			method.Parameters = append(method.Parameters, Parameter{
				Name:        tokens[0],
				Description: tokens[1],
			})
		} else if strings.HasPrefix(line, returnToken) {
			// #nosec
			method.Returns = Returns{
				Description: template.HTML(
					strings.TrimSpace(line[len(returnToken):])),
			}
		} else {
			return false
		}
		return true
	}
	return false
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

func appendClass(classes []Class, cls *Class, method *Method) []Class {
	if method != nil {
		cls.Methods = append(cls.Methods, *method)
	}
	if cls != nil {
		classes = append(classes, *cls)
	}
	return classes
}

func updateDescriptions(line string, context *string, cls *Class, method *Method) {
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
	}
}

func findClasses(blocks [][]string) []Class {
	var classes []Class
	var cls *Class
	var method *Method
	for _, block := range blocks {
		var context *string
		for _, line := range block {
			if strings.HasPrefix(line, classToken) {
				classes = appendClass(classes, cls, method)
				method = nil
				cls = &Class{
					Name: strings.TrimSpace(line[len(classToken):]),
				}
				context = &classToken
				continue
			}
			if cls != nil {
				if strings.HasPrefix(line, methodToken) {
					if method != nil {
						cls.Methods = append(cls.Methods, *method)
					}
					method = &Method{
						Name: strings.TrimSpace(line[len(methodToken):]),
					}
					context = &methodToken
					continue
				}
				if findClassTokens(cls, line) {
					context = nil
				}
			}
			if findMethodTokens(method, line) {
				context = nil
			}
			updateDescriptions(line, context, cls, method)
		}
	}
	classes = appendClass(classes, cls, method)
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
	return findClasses(blocks)
}
