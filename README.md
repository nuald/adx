# adx
Produces the code's auto-generated documentation in HTML, PDF or original XML.

## Requirements

The tool utilizes Doxygen and JsDoc3 command line utilities, please be sure
to install those first, e.g. for macOS:

    $ brew install jsdoc3 doxygen

Please note the tool uses `xsltproc` utility, so be sure that it's available
in PATH.

## Installation

    go get -u github.com/nuald/adx

## Usage

Please use the tool's flags to generate the corresponding output:

```
Usage: adx -lang=(lang) [-src=(src-dir)]+ [-xml=(xml-file)]+ -title=(title) -out=(out.[html|pdf|xml])
Produces the code's auto-generated documentation in HTML, PDF or original XML.

Flags:
  -lang string
      the source code programming language (js, java)
  -out string
      the output file (the format is based on its extension)
  -src value
      the source code dir(s)
  -title string
      the document title
  -xml value
      the input XML file(s)
```
