# adx
Converts the source code's auto-generated documentation to HTML and PDF.

## Requirements

The tool utilizes Doxygen and JsDoc3 command line utilities, please be sure
to install those first, e.g. for macOS:

    $ brew install jsdoc3 doxygen

## Installation

    go get -u github.com/nuald/adx

## Usage

Please use the utility flags to generate the corresponding output:

```
Usage: adx -lang=(lang) -src=(src-dir) -title=(title) -out=(out.[html|pdf])
Converts the source code's auto-generated documentation to HTML and PDF.

Flags:
  -lang string
    	the source code programming language (js, java)
  -out string
    	the output file (the format is based on its extension)
  -src string
    	the source code dir (default ".")
  -title string
    	the document title
```
