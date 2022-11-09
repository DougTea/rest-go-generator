package gin

import (
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/DougTea/go-common/pkg/web"
	"golang.org/x/tools/go/packages"
	"k8s.io/gengo/args"
	"k8s.io/gengo/generator"
	"k8s.io/gengo/namer"
	"k8s.io/gengo/types"
	"k8s.io/klog/v2"
)

const (
	restGinEnabledName = "rest:gin"
	basePath           = "path"
	routePath          = "path"
	method             = "method"
	successCode        = "successCode"
)

var HttpMethods = []web.HttpMethod{
	web.MethodHead,
	web.MethodOptions,
	web.MethodGet,
	web.MethodPost,
	web.MethodPatch,
	web.MethodPut,
	web.MethodDelete,
	web.MethodTrace,
	web.MethodConnect,
}

type addressNamer struct {
	namer.Namer
}

func newAddressNamer(n namer.Namer) *addressNamer {
	return &addressNamer{
		Namer: n,
	}
}

func (n addressNamer) Name(t *types.Type) string {
	if t.Kind == types.Pointer {
		return "&" + n.Namer.Name(t.Elem)
	}
	return n.Namer.Name(t)
}

var routeNameStrategy = &namer.NameStrategy{
	Join: func(pre string, in []string, post string) string {
		res := make([]byte, 0)
		for i := 0; i < len(pre); i++ {
			if c := pre[i]; c >= 'A' && c <= 'Z' {
				res = append(res, '_', 'a'+c-'A')
			} else {
				res = append(res, c)
			}
		}
		in = append(in, post)
		for i := 0; i < len(in); i++ {
			for j := 0; j < len(in[i]); j++ {
				if c := in[i][j]; c >= 'A' && c <= 'Z' {
					res = append(res, '_', 'a'+c-'A')
				} else {
					res = append(res, c)
				}
			}
		}
		return strings.TrimPrefix(string(res), "_")
	},
	IgnoreWords: map[string]bool{
		"Service": true,
		"Biz":     true,
	},
}

func Packages(ctx *generator.Context, arguments *args.GeneratorArgs) generator.Packages {
	header := []byte(fmt.Sprintf("//go:build !%s\n\n", arguments.GeneratedBuildTag))
	pkgs := generator.Packages{}
	for _, i := range arguments.InputDirs {
		klog.V(5).Infof("Considering pkg %q", i)
		pkg := ctx.Universe.Package(i)
		routeGenerators := []generator.Generator{}
		absOutputBase, err := filepath.Abs(arguments.OutputBase)
		if err != nil {
			klog.Fatalf("Error: cannot get abs path of output base,%v", err)
		}
		pkgPath := arguments.OutputPackagePath
		var pkgName, outputPkgPath string
		goPkgs, err := packages.Load(&packages.Config{Mode: packages.NeedModule}, pkg.Path)
		if err != nil || len(goPkgs) != 1 {
			klog.Fatalf("Error: cannot load go package %v", err)
		}
		outputBasePkgs, err := packages.Load(&packages.Config{Mode: packages.NeedName | packages.NeedModule}, arguments.OutputBase)
		if len(outputBasePkgs) == 1 {
			outputPkgPath = filepath.Join(outputBasePkgs[0].PkgPath, arguments.OutputPackagePath)
		}
		if arguments.OutputPackagePath == "" {
			if err == nil && len(outputBasePkgs) == 1 && outputBasePkgs[0].Name != "" {
				pkgName = outputBasePkgs[0].Name
			} else {
				pkgName = filepath.Base(absOutputBase)
			}
		} else {
			pkgName = filepath.Base(arguments.OutputPackagePath)
		}

		for _, t := range pkg.Types {
			if _, ok := types.ExtractCommentTags("+", append(t.CommentLines, t.SecondClosestCommentLines...))[restGinEnabledName]; ok {
				routeGenerators = append(routeGenerators, NewGinGenerator(t, routeNameStrategy.Join("", []string{routeNameStrategy.Name(t)}, arguments.OutputFileBaseName), outputPkgPath))
			}
		}
		if len(routeGenerators) != 0 {
			klog.Infof("Info: generate package name [%v] , path [%v] and outputPkg [%v]", pkgName, pkgPath, outputPkgPath)
			pkgs = append(pkgs, &generator.DefaultPackage{
				PackageName:   pkgName,
				PackagePath:   pkgPath,
				HeaderText:    header,
				GeneratorList: routeGenerators,
			})
		}
	}
	return pkgs
}

type serviceTag struct {
	BasePath string
}

type routeTag struct {
	Method      web.HttpMethod
	Path        string
	SuccessCode int
}

func parseServiceTag(commentLines []string) (*serviceTag, error) {
	values := types.ExtractCommentTags("+", commentLines)
	tagBasePath := "/"
	if v, ok := values[basePath]; ok && len(v) > 0 {
		tagBasePath = v[0]
	}
	return &serviceTag{
		BasePath: tagBasePath,
	}, nil
}

func parseRouteTag(commentLines []string) (*routeTag, error) {
	tag := &routeTag{
		Method:      web.MethodGet,
		Path:        "/",
		SuccessCode: 200,
	}
	values := types.ExtractCommentTags("+", commentLines)
	if m, ok := values[method]; ok && len(m) > 0 {
		for _, httpMethod := range HttpMethods {
			if strings.Contains(strings.ToLower(string(httpMethod)), strings.ToLower(m[0])) {
				tag.Method = httpMethod
				break
			}
		}
	}
	if p, ok := values[routePath]; ok && len(p) > 0 {
		tag.Path = p[0]
	}
	if codes, ok := values[successCode]; ok && len(codes) > 0 {
		code, err := strconv.Atoi(codes[0])
		if err != nil {
			return nil, err
		}
		tag.SuccessCode = code
	}
	return tag, nil
}

type GinGenerator struct {
	generator.DefaultGen
	typeToGenerate *types.Type
	outputPkg      string
	imports        namer.ImportTracker
}

func NewGinGenerator(t *types.Type, name string, outputPkg string) generator.Generator {
	return &GinGenerator{
		DefaultGen:     generator.DefaultGen{OptionalName: name},
		typeToGenerate: t,
		outputPkg:      outputPkg,
		imports:        generator.NewImportTracker(t),
	}
}

func (g *GinGenerator) Namers(c *generator.Context) namer.NameSystems {
	// Have the raw namer for this file track what it imports.
	return namer.NameSystems{
		"public":  namer.NewPublicNamer(0),
		"raw":     namer.NewRawNamer(g.outputPkg, g.imports),
		"address": newAddressNamer(namer.NewRawNamer(g.typeToGenerate.Name.Package, g.imports)),
	}
}

func (g *GinGenerator) Filter(c *generator.Context, t *types.Type) bool {
	return t.Name == g.typeToGenerate.Name
}

func (g *GinGenerator) GenerateType(c *generator.Context, t *types.Type, w io.Writer) error {
	sw := generator.NewSnippetWriter(w, c, "{{", "}}")
	klog.V(5).Infof("processing type %v", t)
	serviceTag, err := parseServiceTag(append(t.CommentLines, t.SecondClosestCommentLines...))
	if err != nil {
		return err
	}
	controllerMap := map[string]interface{}{
		"type": t,
	}

	sw.Do(typeController, controllerMap)
	for k, v := range t.Methods {
		requestType, err := extractRequestType(v)
		if err != nil {
			return err
		}
		responseType, err := extractResponseType(v)
		if err != nil {
			return err
		}
		tag, err := parseRouteTag(append(v.CommentLines, v.SecondClosestCommentLines...))
		if err != nil {
			return err
		}
		httpMethodType := types.Ref("github.com/DougTea/go-common/pkg/web", getStringOfHttpMethod(tag.Method))
		var resultDeclare, requestDeclare string
		operatorDeclare := "="
		if responseType != nil {
			resultDeclare = "r,"
			operatorDeclare = ":="
		}
		if requestType != nil {
			if requestType.Kind != types.Pointer {
				requestDeclare += "*"
			}
			requestDeclare += "p"
		} else if responseType == nil {
			operatorDeclare = ":="
		}
		routeMap := map[string]interface{}{
			"type":           v,
			"serviceType":    t,
			"requestType":    requestType,
			"responseType":   responseType,
			"path":           path.Join(serviceTag.BasePath, tag.Path),
			"funcName":       k,
			"tag":            tag,
			"httpMethodType": httpMethodType,
			"funcInvokeCode": fmt.Sprintf("%serr %s svc.%s(%s)", resultDeclare, operatorDeclare, k, requestDeclare),
		}
		sw.Do(typeGinRoute, routeMap)
	}
	return sw.Error()
}

func getStringOfHttpMethod(m web.HttpMethod) string {
	s := string(m)
	return "Method" + strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

func extractRequestType(t *types.Type) (*types.Type, error) {
	//TODO surport multiple parameters
	if len(t.Signature.Parameters) == 0 {
		return nil, nil
	}
	return t.Signature.Parameters[0], nil
}

func extractResponseType(t *types.Type) (*types.Type, error) {
	//TODO surport multiple parameters
	if len(t.Signature.Results) == 0 || (len(t.Signature.Results) == 1 && t.Signature.Results[0].Name.Name == "error" && t.Signature.Results[0].Name.Package == "") {
		return nil, nil
	}
	return t.Signature.Results[0], nil
}

var typeGinRoute = `
func new{{ .funcName }}RouterOf{{ .serviceType|public }}(svc {{ .serviceType|raw }})*web.Router{
	return &web.Router{
		Method: {{ .httpMethodType|raw }},
		Path: "{{ .path }}",
		Handler: func(c *gin.Context){
			{{- if .requestType }}
			{{- if eq .requestType.Kind "Pointer" }}
			p := new({{ .requestType.Elem|raw }})
			{{- else }}
			p := new({{ .requestType|raw }})
			{{- end }}
			err := c.Bind(p)
			if err!=nil{
				c.Error(err)
				return
			}
			{{- end }}
			{{ .funcInvokeCode }}
			if err!=nil{
				c.Error(err)
			}else{
				{{- if .responseType }}
				c.JSON({{ .tag.SuccessCode }},r)
				{{- else }}
				c.JSON({{ .tag.SuccessCode }},nil)
				{{- end }}
			}
		},
	}
}
`
var typeController = `
type {{ .type|public }}Controller struct{
	service {{ .type|raw }}
	Routers []*web.Router
}

func New{{ .type|public }}Controller(svc {{ .type|raw }})*{{ .type|public }}Controller{
	return &{{ .type|public }}Controller{
		service: svc,
		Routers: []*web.Router{
		{{- range $k,$v := .type.Methods }}
			new{{ $k }}RouterOf{{ $.type|public }}(svc),
		{{- end }}
		},
	}
}
`
