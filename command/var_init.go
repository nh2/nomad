package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/muesli/reflow/wordwrap"
	"github.com/posener/complete"
)

const (
	// DefaultHclVarInitName is the default name we use when initializing the
	// example var file in HCL format
	DefaultHclVarInitName = "spec.nsv.hcl"

	// DefaultHclVarInitName is the default name we use when initializing the
	// example var file in JSON format
	DefaultJsonVarInitName = "spec.nsv.json"
)

// VarInitCommand generates a new secure variable specification
type VarInitCommand struct {
	Meta
}

func (c *VarInitCommand) Help() string {
	helpText := `
Usage: nomad var init <filename>

  Creates an example secure variable specification file that can be used as a
  starting point to customize further. If no filename is given, the default of
  "spec.nsv.hcl" or "spec.nsv.json" will be used.

Init Options:

  -json
    Create an example JSON secure variable specification.

  -q
    Suppress non-error output
`
	return strings.TrimSpace(helpText)
}

func (c *VarInitCommand) Synopsis() string {
	return "Create an example secure variable specification file"
}

func (c *VarInitCommand) AutocompleteFlags() complete.Flags {
	return complete.Flags{
		"-json": complete.PredictNothing,
	}
}

func (c *VarInitCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *VarInitCommand) Name() string { return "var init" }

func (c *VarInitCommand) Run(args []string) int {
	var jsonOutput bool
	var quiet bool

	flags := c.Meta.FlagSet(c.Name(), FlagSetClient)
	flags.Usage = func() { c.Ui.Output(c.Help()) }
	flags.BoolVar(&jsonOutput, "json", false, "")
	flags.BoolVar(&quiet, "q", false, "")

	if err := flags.Parse(args); err != nil {
		return 1
	}

	// Check that we get no arguments
	args = flags.Args()
	if l := len(args); l > 1 {
		c.Ui.Error("This command takes no arguments or one: <filename>")
		c.Ui.Error(commandErrorText(c))
		return 1
	}

	fileName := DefaultHclVarInitName
	fileContent := defaultHclVarSpec
	if jsonOutput {
		fileName = DefaultJsonVarInitName
		fileContent = defaultJsonVarSpec
	}
	if len(args) == 1 {
		fileName = args[0]
	}

	// Check if the file already exists
	_, err := os.Stat(fileName)
	if err != nil && !os.IsNotExist(err) {
		c.Ui.Error(fmt.Sprintf("Failed to stat %q: %v", fileName, err))
		return 1
	}
	if !os.IsNotExist(err) {
		c.Ui.Error(fmt.Sprintf("File %q already exists", fileName))
		return 1
	}

	// Write out the example
	err = ioutil.WriteFile(fileName, []byte(fileContent), 0660)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to write %q: %v", fileName, err))
		return 1
	}

	// Success
	if !quiet {
		c.Ui.Warn(WrapAndPrepend(TidyRawString(msgWarnKeys), 70, ""))
		c.Ui.Output(fmt.Sprintf("Example secure variable specification written to %s", fileName))
	}
	return 0
}

const (
	msgWarnKeys = `
	REMINDER: While keys in the 'Items' collection can contain dots, using
	them in templates is easier when they do not. As a best practice, avoid
	dotted keys when possible.`
)

var defaultHclVarSpec = strings.TrimSpace(`
# A secure variable Path can be specified in the specification file
# and will be used when writing the variable without specifying a
# Path in the command or when writing JSON directly to the `+"`/var/`"+`
# HTTP API endpoint
Path = "path/to/variable" 

# The Namespace to write the variable can be included in the specification
# and is the highest precedence way to set the namespace value.
Namespace = "default"

# The Items collection is the only strictly required part of a secure
# variable specification. It contains the sensitive material to encrypt
# and store as a Nomad secure variable. The entire Items collection are
# encrypted and decrypted as a single unit.

`+warnInHCLFile()+`
Items {
  key1 = "value 1"
  key2 = "value 2"
}
`) + "\n"

var defaultJsonVarSpec = strings.TrimSpace(`
{
  "Items": {
    "key1": "value 1",
    "key2": "value 2"
  }
}
`) + "\n"

func warnInHCLFile() string {
	return WrapAndPrepend(TidyRawString(msgWarnKeys), 70, "# ")
}

// WrapString is a convienience func to abstract away the word wrapping
// implementation
func WrapString(input string, lineLen int) string {
	return wordwrap.String(input, lineLen)
}

// WrapAndPrepend will word wrap the input string to lineLen characters and
// prepend the provided prefix to every line. The total length of each returned
// line will be at most len(input[line])+len(prefix)
func WrapAndPrepend(input string, lineLen int, prefix string) string {
	ss := strings.Split(wordwrap.String(input, lineLen), "\n")
	prefixStringList(ss, prefix)
	return strings.Join(ss, "\n")
}

// HangingIndent will indent all lines except the first line by the
// given indent value, wrapping all lines at lineLen.
func HangingIndent(input string, lineLen int, indent int) string {
	ss := strings.Split(wordwrap.String(input, lineLen), "\n")
	// wrap as though indent was not in play at all and get the first line
	if len(ss) == 1 {
		return ss[0]
	}
	var out strings.Builder
	out.WriteString(ss[0])

	// reassemble and split the wraps taking the indent into account
	input = strings.Join(ss[1:], "\n")
	ss = strings.Split(wordwrap.String(input, lineLen-indent), "\n")
	prefixStringList(ss, strings.Repeat(" ", indent))

	return strings.Join(ss, "\n")
}

// TidyRawString will convert a wrapped and indented raw string into a single
// long string suitable for rewrapping with another tool. It trims leading and
// trailing whitespace and then consume groups of tabs, newlines, and spaces
// replacing them with a single space
func TidyRawString(raw string) string {
	re := regexp.MustCompile("[\t\n ]+")
	return re.ReplaceAllString(strings.TrimSpace(raw), " ")
}

func prefixStringList(ss []string, prefix string) []string {
	for i, s := range ss {
		ss[i] = prefix + s
	}
	return ss
}
