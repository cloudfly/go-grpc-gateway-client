package generator

import (
	"fmt"
	"net/http"
	"strings"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	loopKeyAccessor   = "k"
	loopValueAccessor = "v"

	mapKeyVarName = "key"
)

func getClientInterfaceName(svc *protogen.Service) string {
	return fmt.Sprintf("%sGatewayClient", svc.GoName)
}

func getClientStructName(svc *protogen.Service) string {
	return unexport(getClientInterfaceName(svc))
}

type HTTPRule struct {
	Method  string
	Pattern string
	Body    string
}

func getHTTPRule(m *protogen.Method) (HTTPRule, bool) {
	options, ok := m.Desc.Options().(*descriptorpb.MethodOptions)
	if !ok {
		return HTTPRule{}, false
	}

	rule, ok := proto.GetExtension(options, annotations.E_Http).(*annotations.HttpRule)
	if !ok {
		return HTTPRule{}, false
	}

	switch rule.GetPattern().(type) {
	case *annotations.HttpRule_Get:
		return HTTPRule{
			Method:  http.MethodGet,
			Pattern: rule.GetGet(),
		}, true
	case *annotations.HttpRule_Post:
		return HTTPRule{
			Method:  http.MethodPost,
			Pattern: rule.GetPost(),
			Body:    rule.GetBody(),
		}, true
	case *annotations.HttpRule_Put:
		return HTTPRule{
			Method:  http.MethodPut,
			Pattern: rule.GetPut(),
			Body:    rule.GetBody(),
		}, true
	case *annotations.HttpRule_Patch:
		return HTTPRule{
			Method:  http.MethodPatch,
			Pattern: rule.GetPatch(),
			Body:    rule.GetBody(),
		}, true
	case *annotations.HttpRule_Delete:
		return HTTPRule{
			Method:  http.MethodDelete,
			Pattern: rule.GetDelete(),
			Body:    rule.GetBody(),
		}, true
	default:
		return HTTPRule{}, false
	}
}

func hasGatewayCompatibleMethods(file *protogen.File) bool {
	for _, srv := range file.Services {
		for _, method := range srv.Methods {
			if !isGatewayCompatibleMethod(method) {
				continue
			}
			return true
		}
	}
	return false
}

func isGatewayCompatibleMethod(m *protogen.Method) bool {
	_, ok := getHTTPRule(m)
	return ok && !m.Desc.IsStreamingClient()
}

func generateQueryParam(
	g *protogen.GeneratedFile,
	field *protogen.Field,
	structFields []string,
	isMapKeyDefined bool,
	queryKeyFields ...string,
) {
	isOptional := field.Desc.HasOptionalKeyword()
	isMessage := field.Desc.Message() != nil
	isMap := field.Desc.IsMap()
	isRepeated := field.Desc.Cardinality() == protoreflect.Repeated

	queryKeyName := newStructAccessor(queryKeyFields, field.Desc.JSONName())
	queryValueAccessor := newStructAccessor(structFields, field.GoName)

	// If current field is inside the repeated message, ignore intermediate fields
	// since the loopValueAccessor directs the current field itself.
	if len(structFields) > 1 && structFields[0] == loopValueAccessor {
		queryValueAccessor = newStructAccessor(structFields[:len(structFields)-1], field.GoName)
	}

	if isMap {
		g.P("for ", loopKeyAccessor, ", ", loopValueAccessor, " := range ", queryValueAccessor, " {")
		g.P(mapKeyVarName, " := ", pkgFmt.Ident("Sprintf"), `("`, queryKeyName, `[%v]", `, loopKeyAccessor, ")")
		queryKeyName = mapKeyVarName
		queryValueAccessor = loopValueAccessor
		structFields = []string{loopValueAccessor}
		defer g.P("}")
	} else {
		queryKeyName = fmt.Sprintf("%q", queryKeyName)
		if isRepeated {
			g.P("for _, ", loopValueAccessor, " := range ", queryValueAccessor, " {")
			queryValueAccessor = loopValueAccessor
			structFields = []string{loopValueAccessor}
			defer g.P("}")
		} else if isOptional || isMessage {
			g.P("if ", queryValueAccessor, " != nil {")
			defer g.P("}")
		}
	}

	switch {
	case !isMap && field.Desc.Message() != nil:
		for _, f := range field.Message.Fields {
			generateQueryParam(g, f, append(structFields, field.GoName), isMapKeyDefined, append(queryKeyFields, field.Desc.JSONName())...)
		}
		return
	case field.Desc.Enum() != nil:
		g.P(`q.Add(`, queryKeyName, `, `, queryValueAccessor, ".String())")
	case isOptional:
		g.P(`q.Add(`, queryKeyName, `, `, pkgFmt.Ident("Sprintf"), `("%v", *`, queryValueAccessor, "))")
	default:
		g.P(`q.Add(`, queryKeyName, `, `, pkgFmt.Ident("Sprintf"), `("%v", `, queryValueAccessor, "))")
	}
}

func generateParamValues(g *protogen.GeneratedFile, m *protogen.Method) {
	rule, ok := getHTTPRule(m)
	if !ok {
		return
	}

	g.P(`gwReq, err := c.gwc.NewRequest(ctx, "`, rule.Method, `", "`, rule.Pattern, `")`)
	g.P("if err != nil {")
	g.P(`return nil, fmt.Errorf("new request error: %w", err)`)
	g.P("}")
	fieldsByName := make(map[string]*protogen.Field)

	pathFields := make(map[string]bool)
	queryFields := make(map[string]bool)
	for _, field := range m.Input.Fields {
		fieldName := field.Desc.TextName()
		fieldsByName[fieldName] = field
		if strings.Contains(rule.Pattern, fmt.Sprintf("{%s}", fieldName)) {
			pathFields[fieldName] = true
			valueAccessor := newStructAccessor([]string{"req"}, field.GoName)
			if field.Desc.Enum() != nil {
				g.P(`gwReq = c.gwc.SetPathParam(ctx, gwReq, "`, fieldName, `", `, valueAccessor, ".String())")
			} else {
				g.P(`gwReq = c.gwc.SetPathParam(ctx, gwReq, "`, fieldName, `", `, pkgFmt.Ident("Sprintf"), `("%v", `, valueAccessor, "))")
			}
		} else if rule.Body != "*" && fieldName != rule.Body {
			queryFields[fieldName] = true
		}
	}

	isQueryDefined := false
	for _, field := range m.Input.Fields {
		if _, ok := queryFields[field.Desc.TextName()]; ok {
			if !isQueryDefined {
				g.P("q := ", pkgNetURL.Ident("Values"), "{}")
				isQueryDefined = true
			}
			generateQueryParam(g, field, []string{"req"}, false)
		}
	}
	if isQueryDefined {
		g.P("gwReq.URL.RawQuery = q.Encode()")
	}

	switch rule.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		field := "req"
		if rule.Body != "*" {
			if bodyField, ok := fieldsByName[rule.Body]; ok {
				field = newStructAccessor([]string{field}, bodyField.GoName)
			}
		}
		g.P("body, contentType, err := c.gwc.Marshal(ctx, ", `"`, rule.Method, `"`, ", gwReq.URL.Path, ", field, ")")
		g.P("if err != nil {")
		g.P(`return nil, fmt.Errorf("marshal request error: %w", err)`)
		g.P("}")
		g.P(`if contentType != "" {`)
		g.P(`gwReq.Header.Set("Content-Type", contentType)`)
		g.P(`}`)
		g.P("gwReq.Body = ", pkgIo.Ident("NopCloser"), "(", pkgBytes.Ident("NewBuffer"), "(body)", ")")
	}
}
