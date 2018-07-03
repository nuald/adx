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
Usage: adx [-conf=(yaml-file)] -lang=(lang) [-src=(src-dir)]+ [-xml=(xml-file)]+ -title=(title) -out=(out.[html|pdf|xml])
Produces the code's auto-generated documentation in HTML, PDF or original XML.

Flags:
  -conf string
      the configuration file for the custom languages
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

## Custom Languages Support

Custom languages are parsed based on the configuration files. The code should be documented
in a special way as it's parsed to generate the documentation based on the rules
from the configuration.

Code sample:

```
/**
 * Class: Foo
 * Foo demo class
 *
 * @property prop The sample property.
 * @constructor Build an ThreeDS2Widget class instance.
 */
class Foo(private val prop: String) {

    /**
     * Method: method1
     * The sample method.
     *
     * @param arg The sample argument.
     * @return The sample return.
     */
    fun method1(arg: String): Int {
    }
}
```

*Class:* and *Method:* markers are used to determine the block context. Classes may have
the optional *@constructor* tag to identify that the constructor is implicitly defined
with the *@property* list as its arguments.

The configuration file has the following YAML format:

```
<language_name>:
  docstrings:
    type: [block|line]
    format: /** */
```
