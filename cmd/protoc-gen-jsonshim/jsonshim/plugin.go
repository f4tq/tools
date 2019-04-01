package jsonshim

import (
	"path"
	"strings"

	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
)

var _ = gogoproto.E_Benchgen

func init() {
	generator.RegisterPlugin(NewPlugin())
}

// FileNameSuffix is the suffix added to files generated by jsonshim
const FileNameSuffix = "_json_generated.go"

// Plugin is a protoc-gen-gogo plugin that creates MarshalJSON() and
// UnmarshalJSON() functions for protobuf types that use oneof fields.
type Plugin struct {
	*generator.Generator
	generator.PluginImports
	filesWritten map[string]interface{}
}

// NewPlugin returns a new instance of the Plugin
func NewPlugin() *Plugin {
	return &Plugin{
		filesWritten: map[string]interface{}{},
	}
}

// Name returns the name of this plugin
func (p *Plugin) Name() string {
	return "jsonshim"
}

// Init initializes our plugin with the active generator
func (p *Plugin) Init(g *generator.Generator) {
	p.Generator = g
}

var (
	wkt = []string{
		".google.protobuf.Duration",
		".google.protobuf.Struct",
		".google.protobuf.BoolValue",
	}
)
// Generate our content
func (p *Plugin) Generate(file *generator.FileDescriptor) {
	p.PluginImports = generator.NewPluginImports(p.Generator)

	// imported packages
	bytesPkg := p.NewImport("bytes")
	jsonpbPkg := p.NewImport("github.com/gogo/protobuf/jsonpb")

	wroteMarshalers := false
	marshalerName := generator.FileName(file) + "Marshaler"
	unmarshalerName := generator.FileName(file) + "Unmarshaler"
	for _, message := range file.Messages() {
		// check to make sure something was generated for this type
		p.P(`// message: `, message.Name )

		if !gogoproto.HasTypeDecl(file.FileDescriptorProto, message.DescriptorProto)  {
			continue
		}
		if  message.GetOptions().GetMapEntry()  {
			p.P(`// skipping message: `, message.Name )
			continue
		}
		p.P(`// Generating Marshal for message: `, message.Name )

		typeName := generator.CamelCaseSlice(message.TypeName())

		// Generate MarshalJSON() method for this type
		p.P(`// MarshalJSON is a custom marshaler supporting oneof fields for `, typeName)
		p.P(`func (this *`, typeName, `) MarshalJSON() ([]byte, error) {`)
		p.In()
		p.P(`str, err := `, marshalerName, `.MarshalToString(this)`)
		p.P(`return []byte(str), err`)
		p.Out()
		p.P(`}`)

		// Generate UnmarshalJSON() method for this type
		p.P(`// UnmarshalJSON is a custom unmarshaler supporting oneof fields for `, typeName)
		p.P(`func (this *`, typeName, `) UnmarshalJSON(b []byte) error {`)
		p.In()
		p.P(`return `, unmarshalerName, `.Unmarshal(`, bytesPkg.Use(), `.NewReader(b), this)`)
		p.Out()
		p.P(`}`)

		wroteMarshalers = true
	}

	if !wroteMarshalers {
		return
	}

	// write out globals
	p.P(`var (`)
	p.In()
	p.P(marshalerName, ` = &`, jsonpbPkg.Use(), `.Marshaler{}`)
	p.P(unmarshalerName, ` = &`, jsonpbPkg.Use(), `.Unmarshaler{}`)
	p.Out()
	p.P(`)`)

	// store this file away
	p.addFile(file)
}

func (p *Plugin) addFile(file *generator.FileDescriptor) {
	name := file.GetName()
	importPath := ""
	// the relevant bits of FileDescriptor.goPackageOption(), if only it were exported
	opt := file.GetOptions().GetGoPackage()
	if opt != "" {
		if sc := strings.Index(opt, ";"); sc >= 0 {
			// A semicolon-delimited suffix delimits the import path and package name.
			importPath = opt[:sc]
		} else if strings.LastIndex(opt, "/") > 0 {
			// The presence of a slash implies there's an import path.
			importPath = opt
		}
	}
	// strip the extension
	name = name[:len(name)-len(path.Ext(name))]
	if importPath != "" {
		name = path.Join(importPath, path.Base(name))
	}
	p.filesWritten[name+FileNameSuffix] = struct{}{}
}

// FilesWritten returns a list of the names of files for which output was generated
func (p *Plugin) FilesWritten() map[string]interface{} {
	return p.filesWritten
}


/*
if (len(message.GetOneofDecl()) == 0 && len(message.GetEnumType()) == 0 && len(message.GetNestedType())==0) || message.GetOptions().GetMapEntry()  {
	p.P(`// skipping message: `, message.Name )
	continue
}
*/
/*
if !gogoproto.HasTypeDecl(file.FileDescriptorProto, message.DescriptorProto) ||
	(len(message.GetOneofDecl()) == 0 && len(message.GetEnumType()) == 0) {
	googleAny := false
Loop:
	for _, field := range message.Field {
		p.P(`// fieldName `, field.String())
		if field.TypeName == nil { continue }
		for _, extension := range wkt {
			if  strings.HasPrefix(extension, *field.TypeName) {
				p.P(`// message has wkt `, message.Name )
				googleAny = true
				break Loop
			}
		}
	}
	if !googleAny {
		continue
	}
}
*/
