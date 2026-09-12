/*
 *  MIT License
 *
 * Copyright (c) 2025 Jonas Kaninda
 *
 *  Permission is hereby granted, free of charge, to any person obtaining a copy
 *  of this software and associated documentation files (the "Software"), to deal
 *  in the Software without restriction, including without limitation the rights
 *  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 *  copies of the Software, and to permit persons to whom the Software is
 *  furnished to do so, subject to the following conditions:
 *
 *  The above copyright notice and this permission notice shall be included in all
 *  copies or substantial portions of the Software.
 *
 *  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 *  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 *  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 *  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 *  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 *  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 *  SOFTWARE.
 */

package okapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/jkaninda/njia"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	schemaRefPrefix = "#/components/schemas/"
	specTestLicense = "MIT"
	specTestServer  = "http://localhost:8080"
)

// recursiveNode refers to itself through a slice.
type recursiveNode struct {
	Name     string          `json:"name"`
	Children []recursiveNode `json:"children"`
}

// recursiveList refers to itself through a pointer.
type recursiveList struct {
	Value int            `json:"value"`
	Next  *recursiveList `json:"next" description:"Next element"`
}

// recursiveMap refers to itself through a map value.
type recursiveMap struct {
	Label    string                  `json:"label"`
	Branches map[string]recursiveMap `json:"branches"`
}

// mutualA and mutualB refer to each other.
type mutualA struct {
	ID string    `json:"id"`
	Bs []mutualB `json:"bs"`
}

type mutualB struct {
	ID string   `json:"id"`
	A  *mutualA `json:"a"`
}

// RecursiveBase is only ever reachable through recursiveEmbedder, so its
// component has to be created from the recursive reference alone.
type RecursiveBase struct {
	Parent *RecursiveBase `json:"parent"`
}

type recursiveEmbedder struct {
	RecursiveBase
	Title string `json:"title"`
}

// newSpecTestApp returns an app whose documents pass validation apart from the
// routes under test (a license name and a server URL are required).
func newSpecTestApp(title string) *Okapi {
	o := New()
	o.WithOpenAPIDocs(OpenAPI{
		Title:   title,
		Version: defaultAPIVersion,
		License: License{Name: specTestLicense},
		Servers: Servers{{URL: specTestServer}},
	})
	return o
}

// forEachSpec runs fn as a subtest, named after the version, for the 3.0 and
// the 3.1 document of o.
func forEachSpec(t *testing.T, o *Okapi, fn func(t *testing.T, spec *openapi3.T)) {
	t.Helper()
	for _, spec := range []*openapi3.T{o.openapiSpec, o.openapiSpec31} {
		t.Run(spec.OpenAPI, func(t *testing.T) { fn(t, spec) })
	}
}

// componentSchema returns the named component schema, failing the test if absent.
func componentSchema(t *testing.T, spec *openapi3.T, name string) *openapi3.Schema {
	t.Helper()
	ref, ok := spec.Components.Schemas[name]
	require.Truef(t, ok, "component %q missing", name)
	require.NotNil(t, ref.Value)
	return ref.Value
}

// requireComponentRefs asserts every $ref in the marshalled document names an
// existing component.
func requireComponentRefs(t *testing.T, spec *openapi3.T) {
	t.Helper()
	data, err := spec.MarshalJSON()
	require.NoError(t, err)
	var doc any
	require.NoError(t, json.Unmarshal(data, &doc))
	var check func(v any)
	check = func(v any) {
		switch v := v.(type) {
		case map[string]any:
			if ref, ok := v["$ref"].(string); ok {
				require.True(t, strings.HasPrefix(ref, schemaRefPrefix), ref)
				require.Contains(t, spec.Components.Schemas, strings.TrimPrefix(ref, schemaRefPrefix), "dangling %s", ref)
			}
			for _, child := range v {
				check(child)
			}
		case []any:
			for _, child := range v {
				check(child)
			}
		}
	}
	check(doc)
}

// fieldRef returns the component reference of a recursive field, which is
// wrapped so it can carry field-level keywords such as nullable.
func fieldRef(t *testing.T, s *openapi3.Schema) string {
	t.Helper()
	require.NotNil(t, s)
	refs := s.AllOf
	if len(refs) == 0 {
		refs = s.AnyOf
	}
	require.NotEmpty(t, refs)
	return refs[0].Ref
}

func TestOpenAPIRecursiveSchemas(t *testing.T) {
	o := newSpecTestApp("Recursive")
	o.Get("/tree", anyHandler, DocResponse(recursiveNode{}))
	o.Get("/list", anyHandler, DocResponse(&recursiveList{}))
	o.Put("/map", anyHandler, DocRequestBody(recursiveMap{}), DocResponse(recursiveMap{}))
	o.Post("/mutual", anyHandler, DocRequestBody(mutualA{}), DocResponse(mutualB{}))
	o.Get("/embedded", anyHandler, DocResponse(recursiveEmbedder{}))
	o.Webhook("treeChanged", http.MethodPost, DocRequestBody(recursiveNode{}))
	o.buildOpenAPISpec()

	forEachSpec(t, o, func(t *testing.T, spec *openapi3.T) {
		node := componentSchema(t, spec, "recursiveNode")
		assert.Equal(t, schemaRefPrefix+"recursiveNode", node.Properties["children"].Value.Items.Ref)

		branches := componentSchema(t, spec, "recursiveMap").Properties["branches"].Value
		assert.Equal(t, schemaRefPrefix+"recursiveMap", branches.AdditionalProperties.Schema.Ref)

		// mutualA inlines mutualB, whose back-reference points at mutualA.
		bItems := componentSchema(t, spec, "mutualA").Properties["bs"].Value.Items.Value
		require.NotNil(t, bItems)
		assert.Equal(t, schemaRefPrefix+"mutualA", fieldRef(t, bItems.Properties["a"].Value))
		// mutualB inlines mutualA, whose back-reference points at mutualB.
		aInline := componentSchema(t, spec, "mutualB").Properties["a"].Value
		assert.Equal(t, schemaRefPrefix+"mutualB", aInline.Properties["bs"].Value.Items.Ref)

		// The embedded type's component exists although no route uses it directly.
		componentSchema(t, spec, "RecursiveBase")
		parent := componentSchema(t, spec, "recursiveEmbedder").Properties["parent"].Value
		assert.Equal(t, schemaRefPrefix+"RecursiveBase", fieldRef(t, parent))

		requireComponentRefs(t, spec)
		validateOpenAPIDoc(t, spec)
	})

	// Pointer self-references stay nullable and keep field-level keywords.
	next30 := componentSchema(t, o.openapiSpec, "recursiveList").Properties["next"].Value
	assert.True(t, next30.Nullable)
	assert.Equal(t, "Next element", next30.Description)
	require.Len(t, next30.AllOf, 1)
	assert.Equal(t, schemaRefPrefix+"recursiveList", next30.AllOf[0].Ref)

	next31 := componentSchema(t, o.openapiSpec31, "recursiveList").Properties["next"].Value
	assert.False(t, next31.Nullable)
	assert.Equal(t, "Next element", next31.Description)
	assert.Empty(t, next31.AllOf)
	require.Len(t, next31.AnyOf, 2)
	assert.Equal(t, schemaRefPrefix+"recursiveList", next31.AnyOf[0].Ref)
	assert.True(t, next31.AnyOf[1].Value.Type.Is(openapi3.TypeNull))

	// Webhook payloads (3.1 only) resolve against the 3.1 components.
	require.Contains(t, o.openapiSpec31.Webhooks, "treeChanged")

	// Rebuilding produces the same references.
	o.buildOpenAPISpec()
	forEachSpec(t, o, func(t *testing.T, spec *openapi3.T) {
		requireComponentRefs(t, spec)
		validateOpenAPIDoc(t, spec)
	})
}

// TestOpenAPIRecursiveSchemaNameConflict checks that recursive references follow
// the component deduplication: when the type name is already taken by a
// different schema, the reference points at the suffixed component.
func TestOpenAPIRecursiveSchemaNameConflict(t *testing.T) {
	o := newSpecTestApp("Recursive conflict")
	taken := openapi3.NewObjectSchema().WithProperty("other", openapi3.NewStringSchema())
	taken.Title = "recursiveNode"
	require.NoError(t, o.RegisterSchemas(map[string]*SchemaInfo{
		"recursiveNode": {Schema: openapi3.NewSchemaRef("", taken)},
	}))
	o.Get("/tree", anyHandler, DocResponse(recursiveNode{}))
	o.buildOpenAPISpec()

	forEachSpec(t, o, func(t *testing.T, spec *openapi3.T) {
		assert.Contains(t, componentSchema(t, spec, "recursiveNode").Properties, "other")
		node := componentSchema(t, spec, "recursiveNode1")
		assert.Equal(t, schemaRefPrefix+"recursiveNode1", node.Properties["children"].Value.Items.Ref)
		requireComponentRefs(t, spec)
		validateOpenAPIDoc(t, spec)
	})
}

func TestOpenAPIAnyRoute(t *testing.T) {
	o := newSpecTestApp("Any")
	o.addRoute(njia.MethodAny, "/proxy", nil, anyHandler, DocOperationId("proxy"), DocResponse(nullable31Model{}))
	o.addRoute(njia.MethodAny, "/forward", nil, anyHandler, DocSummary("Forward everything"))
	o.addRoute(njia.MethodAny, "/anonymous", nil, anyHandler)
	// An explicitly registered method wins over the wildcard, whatever the order.
	o.Get("/files", anyHandler, DocOperationId("readFile"))
	o.addRoute(njia.MethodAny, "/files", nil, anyHandler, DocOperationId("files"))
	o.addRoute(njia.MethodAny, "/items", nil, anyHandler, DocOperationId("items"))
	o.Delete("/items", anyHandler, DocOperationId("deleteItem"))
	o.Webhook("anyEvent", njia.MethodAny, DocOperationId("anyEvent"))
	o.buildOpenAPISpec()

	// The router accepts and serves the wildcard route.
	for _, method := range []string{http.MethodGet, http.MethodPut, "PROPFIND"} {
		rec := httptest.NewRecorder()
		o.ServeHTTP(rec, httptest.NewRequest(method, "/proxy", nil))
		assert.Equal(t, http.StatusOK, rec.Code, method)
	}

	anyMethods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	forEachSpec(t, o, func(t *testing.T, spec *openapi3.T) {
		for path, id := range map[string]string{"/proxy": "proxy", "/forward": "forward-everything", "/anonymous": ""} {
			item := spec.Paths.Value(path)
			require.NotNil(t, item, path)
			assert.Len(t, item.Operations(), len(anyMethods), path)
			for _, method := range anyMethods {
				op := item.GetOperation(method)
				require.NotNil(t, op, "%s %s", method, path)
				if id == "" {
					assert.Empty(t, op.OperationID)
				} else {
					assert.Equal(t, id+"-"+strings.ToLower(method), op.OperationID)
				}
			}
		}
		assert.Contains(t, spec.Paths.Value("/proxy").Get.Responses.Map(), "200")

		files := spec.Paths.Value("/files")
		assert.Equal(t, "readFile", files.Get.OperationID)
		assert.Equal(t, "files-post", files.Post.OperationID)
		items := spec.Paths.Value("/items")
		assert.Equal(t, "deleteItem", items.Delete.OperationID)
		assert.Equal(t, "items-get", items.Get.OperationID)

		for path, item := range spec.Paths.Map() {
			assert.NotEmpty(t, item.Operations(), "empty path item %s", path)
		}
		validateOpenAPIDoc(t, spec)
	})

	webhook := o.openapiSpec31.Webhooks["anyEvent"]
	require.NotNil(t, webhook)
	assert.Len(t, webhook.Operations(), len(anyMethods))
	assert.Equal(t, "anyEvent-put", webhook.Put.OperationID)
}
