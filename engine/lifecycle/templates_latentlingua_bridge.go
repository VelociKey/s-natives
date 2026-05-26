package lifecycle

const BridgeGo = `package bridge

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sov.fleet/s-latentlingua/02000-logic-libraries/snparser"
)

// ToProto converts a parsed WEBNF grammar into a Protobuf definition.
func ToProto(grammar *snparser.GrammarNode) (string, error) {
	var sb strings.Builder
	sb.WriteString("syntax = \"proto3\";\n\n")

	// Iterate over rules to find enums first
	for name, rule := range grammar.Rules {
		if choice, ok := rule.Expr.(*snparser.ChoiceNode); ok {
			sb.WriteString(fmt.Sprintf("enum %s {\n", name))
			for i, opt := range choice.Options {
				if term, ok := opt.(*snparser.TerminalNode); ok {
					val := strings.Trim(term.Value, "\"")
					sb.WriteString(fmt.Sprintf("\t%s = %d;\n", val, i))
				}
			}
			sb.WriteString("}\n\n")
		}
	}

	// Iterate over rules to find messages
	for name, rule := range grammar.Rules {
		if seq, ok := rule.Expr.(*snparser.SequenceNode); ok {
			// Check if it's a FlatBlock (sequence of RuleNodes)
			isMessage := true
			for _, el := range seq.Elements {
				if _, ok := el.(*snparser.RuleNode); !ok {
					isMessage = false
					break
				}
			}

			if isMessage {
				sb.WriteString(fmt.Sprintf("message %s {\n", name))
				for i, el := range seq.Elements {
					rn := el.(*snparser.RuleNode)
					typ, repeated := mapWEBNFType(rn.Expr)
					protoTyp := mapToProtoType(typ)
					prefix := ""
					if repeated { prefix = "repeated " }
					sb.WriteString(fmt.Sprintf("\t%s%s %s = %d;\n", prefix, protoTyp, rn.Name, i+1))
				}
				sb.WriteString("}\n\n")
			}
		}
	}

	return sb.String(), nil
}

func mapWEBNFType(node snparser.Node) (string, bool) {
	switch n := node.(type) {
	case *snparser.TerminalNode:
		return n.Value, false
	case *snparser.RepetitionNode:
		typ, isRepeated := mapWEBNFType(n.Expr)
		if isRepeated {
			// already repeated
		}
		return typ, true
	default:
		return "string", false
	}
}

func mapToProtoType(w string) string {
	switch w {
	case "string": return "string"
	case "number": return "int64"
	case "boolean": return "bool"
	default: return w
	}
}

// FromProto converts a Protobuf definition string into WEBNF.
func FromProto(proto string) (string, error) {
	var sb strings.Builder
	sb.WriteString(":Sovereign:Imported:Proto:v1\n\n")

	// Match enums
	enumRegex := regexp.MustCompile("enum\\s+(\\w+)\\s*\\{([^}]+)\\}")
	enums := enumRegex.FindAllStringSubmatch(proto, -1)
	for _, m := range enums {
		name := m[1]
		body := m[2]
		sb.WriteString(fmt.Sprintf("%s = ", name))
		
		valRegex := regexp.MustCompile("(\\w+)\\s*=\\s*\\d+;")
		vals := valRegex.FindAllStringSubmatch(body, -1)
		var choices []string
		for _, v := range vals {
			choices = append(choices, fmt.Sprintf("\"%s\"", v[1]))
		}
		sb.WriteString(strings.Join(choices, " / "))
		sb.WriteString(" ;\n\n")
	}

	// Match messages
	msgRegex := regexp.MustCompile("message\\s+(\\w+)\\s*\\{([^}]+)\\}")
	msgs := msgRegex.FindAllStringSubmatch(proto, -1)
	for _, m := range msgs {
		name := m[1]
		body := m[2]
		sb.WriteString(fmt.Sprintf("%s = {\n", name))
		
		fieldRegex := regexp.MustCompile("(?:repeated\\s+)?(\\w+)\\s+(\\w+)\\s*=\\s*\\d+;")
		fields := fieldRegex.FindAllStringSubmatch(body, -1)
		for _, f := range fields {
			typ := mapProtoType(f[1])
			isRepeated := strings.Contains(f[0], "repeated")
			if isRepeated {
				sb.WriteString(fmt.Sprintf("\t%s = { %s } ;\n", f[2], typ))
			} else {
				sb.WriteString(fmt.Sprintf("\t%s = %s ;\n", f[2], typ))
			}
		}
		sb.WriteString("}\n\n")
	}

	return sb.String(), nil
}

// FromStruct uses reflection to convert a Go struct into WEBNF.
func FromStruct(v interface{}) (string, error) {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return "", fmt.Errorf("FromStruct requires a struct input")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(":Sovereign:Imported:Struct:%s:v1\n\n", t.Name()))
	sb.WriteString(fmt.Sprintf("%s = {\n", t.Name()))

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		typ := mapGoType(field.Type)
		sb.WriteString(fmt.Sprintf("\t%s = %s ;\n", field.Name, typ))
	}
	sb.WriteString("}\n")

	return sb.String(), nil
}

func mapProtoType(p string) string {
	switch p {
	case "string": return "string"
	case "int32", "int64", "uint32", "uint64": return "number"
	case "bool": return "boolean"
	default: return p // Assume it's another message or enum
	}
}

func mapGoType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String: return "string"
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64: return "number"
	case reflect.Bool: return "boolean"
	case reflect.Slice: return fmt.Sprintf("{ %s }", mapGoType(t.Elem()))
	case reflect.Struct: return t.Name()
	default: return "any"
	}
}
`
