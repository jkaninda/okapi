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
	"sort"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// JSON member names asserted by several embedding tests.
const (
	memberName      = "name"
	memberEntity    = "entity"
	memberUpdatedBy = "updated_by"
	memberEmail     = "contact_email"
)

// specTestConfig is a documentation config whose documents pass validation
// apart from the routes under test.
func specTestConfig(title string) OpenAPI {
	return OpenAPI{
		Title:   title,
		Version: defaultAPIVersion,
		License: License{Name: specTestLicense},
		Servers: Servers{{URL: specTestServer}},
	}
}

// specDocument marshals spec and decodes it into generic JSON values, as a
// client of the served document sees it.
func specDocument(t *testing.T, spec *openapi3.T) map[string]any {
	t.Helper()
	data, err := spec.MarshalJSON()
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(data, &doc))
	return doc
}

// jsonObject follows keys from v and returns the JSON object found there.
func jsonObject(t *testing.T, v any, keys ...string) map[string]any {
	t.Helper()
	for _, key := range keys {
		obj, ok := v.(map[string]any)
		require.Truef(t, ok, "no object holding %q", key)
		v = obj[key]
	}
	obj, ok := v.(map[string]any)
	require.Truef(t, ok, "%v is not an object", keys)
	return obj
}

// documentProperties returns the properties of a component schema of a
// decoded document.
func documentProperties(t *testing.T, doc map[string]any, component string) map[string]any {
	t.Helper()
	return jsonObject(t, doc, "components", "schemas", component, "properties")
}

// documentExample returns the example of a decoded schema: `example` in 3.0,
// the single entry of `examples` in 3.1.
func documentExample(t *testing.T, doc map[string]any, schema map[string]any) any {
	t.Helper()
	if doc["openapi"] == openApiVersion {
		assert.NotContains(t, schema, "examples")
		return schema["example"]
	}
	assert.NotContains(t, schema, "example")
	examples, ok := schema["examples"].([]any)
	require.True(t, ok, "examples missing")
	require.Len(t, examples, 1)
	return examples[0]
}

type exampleCoordinates struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// typedExampleModel carries example tags on fields of every JSON type.
type typedExampleModel struct {
	ID        int                `json:"id" example:"1"`
	Count     uint8              `json:"count" example:"7"`
	Big       int64              `json:"big" example:"-9007199254740991"`
	Price     float32            `json:"unit_price" example:"9.5"`
	Active    bool               `json:"is_active" example:"true"`
	Parent    *int               `json:"parent" example:"3"`
	Enabled   *bool              `json:"enabled" example:"false"`
	Name      string             `json:"display_name" example:"42"`
	Tags      []string           `json:"tag_list" example:"[\"a\",\"b\"]"`
	Labels    map[string]string  `json:"labels" example:"{\"env\":\"prod\"}"`
	Location  exampleCoordinates `json:"location" example:"{\"lat\":1.5,\"lng\":2}"`
	Metadata  any                `json:"metadata" example:"{\"origin\":\"sensor-feed\"}"`
	Note      any                `json:"note" example:"free text"`
	CreatedOn time.Time          `json:"created_on" example:"2025-01-02T15:04:05Z"`
}

// untypedExampleModel carries example tags that cannot be read as the field's
// type; they are kept as strings.
type untypedExampleModel struct {
	ID    int       `json:"id" example:"eleven"`
	Small uint8     `json:"small" example:"300"`
	Ratio float64   `json:"ratio" example:"NaN"`
	Flag  bool      `json:"flag_value" example:"maybe"`
	IDs   []int     `json:"ids" example:"1,2"`
	Year  time.Time `json:"year" example:"2025"`
}

func TestOpenAPITypedExamples(t *testing.T) {
	o := newSpecTestApp("Typed examples")
	o.Post("/examples", anyHandler, DocRequestBody(typedExampleModel{}), DocResponse(typedExampleModel{}))
	o.Put("/untyped", anyHandler, DocRequestBody(untypedExampleModel{}))
	o.buildOpenAPISpec()

	forEachSpec(t, o, func(t *testing.T, spec *openapi3.T) {
		doc := specDocument(t, spec)
		props := documentProperties(t, doc, "typedExampleModel")
		expected := map[string]any{
			"id":           float64(1),
			"count":        float64(7),
			"big":          float64(-9007199254740991),
			"unit_price":   9.5,
			"is_active":    true,
			"parent":       float64(3),
			"enabled":      false,
			"display_name": "42",
			"tag_list":     []any{"a", "b"},
			"labels":       map[string]any{"env": "prod"},
			"location":     map[string]any{"lat": 1.5, "lng": float64(2)},
			"metadata":     map[string]any{"origin": "sensor-feed"},
			"note":         "free text",
			"created_on":   "2025-01-02T15:04:05Z",
		}
		for name, want := range expected {
			assert.Equal(t, want, documentExample(t, doc, jsonObject(t, props, name)), name)
		}

		untyped := documentProperties(t, doc, "untypedExampleModel")
		for name, want := range map[string]string{
			"id": "eleven", "small": "300", "ratio": "NaN", "flag_value": "maybe", "ids": "1,2", "year": "2025",
		} {
			assert.Equal(t, want, documentExample(t, doc, jsonObject(t, untyped, name)), name)
		}
	})

	// Only the typed examples are valid against their schemas.
	valid := newSpecTestApp("Typed examples")
	valid.Post("/examples", anyHandler, DocRequestBody(&typedExampleModel{}), DocResponse(typedExampleModel{}))
	valid.buildOpenAPISpec()
	forEachSpec(t, valid, func(t *testing.T, spec *openapi3.T) {
		validateOpenAPIDoc(t, spec)
	})
}

// embeddedEntity is unexported; encoding/json still promotes its exported fields.
type embeddedEntity struct {
	ID        string `json:"id" validate:"required"`
	Name      string `json:"name" validate:"required"`
	CreatedBy string `json:"created_by,omitempty"`
	revision  int
}

// EmbeddedAudit is exported and embedded by pointer.
type EmbeddedAudit struct {
	UpdatedBy string `json:"updated_by" required:"true"`
}

type embeddedTimestamps struct {
	UpdatedAt time.Time `json:"updated_at"`
}

// embeddedCode is an unexported non-struct type, ignored when embedded.
type embeddedCode string

// EmbeddedLabel is an exported non-struct type, a property named after the
// type when embedded.
type EmbeddedLabel string

type embeddedAccount struct {
	embeddedEntity
	*EmbeddedAudit
	*embeddedTimestamps
	embeddedCode
	EmbeddedLabel
	// Name shadows embeddedEntity.Name, which is required while Name is not.
	Name  string `json:"name,omitempty"`
	Email string `json:"contact_email" validate:"required"`
}

// embeddedNested embeds structs under a JSON name, which nests them.
type embeddedNested struct {
	embeddedEntity      `json:"entity" validate:"required"`
	EmbeddedAudit       `json:"audit,omitempty"`
	*embeddedTimestamps `json:"-"`
	Title               string `json:"heading"`
}

type embeddedLeft struct {
	Code string
	Left string `json:"left"`
}

type embeddedRight struct {
	Code  string
	Right string `json:"right"`
}

type embeddedDeep struct {
	embeddedLeft
	Depth int `json:"depth"`
}

type EmbeddedTaggedKind struct {
	Value string `json:"Kind"`
}

type embeddedUntaggedKind struct {
	Kind string
}

type embeddedHidden struct {
	Secret string `json:"secret"`
}

// embeddedConflicts follows the encoding/json dominance rules: Code is
// ambiguous at the shallowest depth and dropped, although embeddedDeep has it
// deeper; the tagged Kind beats the untagged one at the same depth.
type embeddedConflicts struct {
	embeddedLeft
	embeddedRight
	embeddedDeep
	EmbeddedTaggedKind
	embeddedUntaggedKind
	embeddedHidden `hidden:"true"`
}

// embeddedRecursiveBase is unexported and only reachable through embedding.
type embeddedRecursiveBase struct {
	Parent *embeddedRecursiveBase `json:"parent"`
}

type embeddedRecursive struct {
	embeddedRecursiveBase
	Title string `json:"heading"`
}

func TestOpenAPIEmbeddedStructs(t *testing.T) {
	o := newSpecTestApp("Embedded")
	o.Get("/accounts", anyHandler, DocResponse(embeddedAccount{}))
	o.Get("/nested", anyHandler, DocResponse(&embeddedNested{}))
	o.Get("/conflicts", anyHandler, DocResponse(embeddedConflicts{}))
	o.Get("/recursive", anyHandler, DocResponse(embeddedRecursive{}))
	o.buildOpenAPISpec()

	forEachSpec(t, o, func(t *testing.T, spec *openapi3.T) {
		account := componentSchema(t, spec, "embeddedAccount")
		assert.ElementsMatch(t, []string{"id", memberName, "created_by", memberUpdatedBy, "updated_at", "EmbeddedLabel", memberEmail},
			propertyNames(account))
		assert.Equal(t, []string{"id", memberUpdatedBy, memberEmail}, account.Required)

		nested := componentSchema(t, spec, "embeddedNested")
		assert.ElementsMatch(t, []string{memberEntity, "audit", "heading"}, propertyNames(nested))
		assert.Equal(t, []string{memberEntity}, nested.Required)
		require.Contains(t, nested.Properties, memberEntity)
		entity := nested.Properties[memberEntity].Value
		require.NotNil(t, entity)
		assert.ElementsMatch(t, []string{"id", memberName, "created_by"}, propertyNames(entity))
		assert.Equal(t, []string{"id", memberName}, entity.Required)
		require.Contains(t, nested.Properties, "audit")
		assert.ElementsMatch(t, []string{memberUpdatedBy}, propertyNames(nested.Properties["audit"].Value))

		conflicts := componentSchema(t, spec, "embeddedConflicts")
		assert.ElementsMatch(t, []string{"left", "right", "depth", "Kind"}, propertyNames(conflicts))

		recursive := componentSchema(t, spec, "embeddedRecursive")
		assert.ElementsMatch(t, []string{"parent", "heading"}, propertyNames(recursive))
		require.Contains(t, recursive.Properties, "parent")
		componentSchema(t, spec, "embeddedRecursiveBase")
		assert.Equal(t, schemaRefPrefix+"embeddedRecursiveBase", fieldRef(t, recursive.Properties["parent"].Value))

		requireComponentRefs(t, spec)
		validateOpenAPIDoc(t, spec)
	})
}

// TestOpenAPIEmbeddedMatchesEncodingJSON checks that the schema documents
// exactly the members encoding/json writes.
func TestOpenAPIEmbeddedMatchesEncodingJSON(t *testing.T) {
	values := []any{
		embeddedAccount{
			embeddedEntity:     embeddedEntity{ID: "1", Name: "inner", CreatedBy: "me", revision: 1},
			EmbeddedAudit:      &EmbeddedAudit{UpdatedBy: "me"},
			embeddedTimestamps: &embeddedTimestamps{UpdatedAt: time.Now()},
			embeddedCode:       "xy",
			EmbeddedLabel:      "label",
			Name:               "outer",
			Email:              "me@example.com",
		},
		embeddedNested{
			embeddedEntity:     embeddedEntity{ID: "1", Name: "nested", CreatedBy: "me"},
			EmbeddedAudit:      EmbeddedAudit{UpdatedBy: "me"},
			embeddedTimestamps: &embeddedTimestamps{},
			Title:              "t",
		},
		embeddedConflicts{
			embeddedLeft:         embeddedLeft{Code: "l", Left: "l"},
			embeddedRight:        embeddedRight{Code: "r", Right: "r"},
			embeddedDeep:         embeddedDeep{embeddedLeft: embeddedLeft{Code: "d"}, Depth: 2},
			EmbeddedTaggedKind:   EmbeddedTaggedKind{Value: "tagged"},
			embeddedUntaggedKind: embeddedUntaggedKind{Kind: "untagged"},
			embeddedHidden:       embeddedHidden{Secret: "hidden"},
		},
		embeddedRecursive{embeddedRecursiveBase: embeddedRecursiveBase{Parent: &embeddedRecursiveBase{}}, Title: "t"},
	}
	for _, v := range values {
		data, err := json.Marshal(v)
		require.NoError(t, err)
		var members map[string]any
		require.NoError(t, json.Unmarshal(data, &members))
		names := make([]string, 0, len(members))
		for name := range members {
			names = append(names, name)
		}
		schema := reflectToSchemaWithInfo(v).Schema.Value
		if _, ok := v.(embeddedConflicts); ok {
			// The hidden embedded struct is serialized but not documented.
			names = removeString(names, "secret")
		}
		assert.ElementsMatch(t, names, propertyNames(schema), "%T", v)
	}
}

func propertyNames(s *openapi3.Schema) []string {
	names := make([]string, 0, len(s.Properties))
	for name := range s.Properties {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func removeString(values []string, value string) []string {
	out := values[:0]
	for _, v := range values {
		if v != value {
			out = append(out, v)
		}
	}
	return out
}

// TestOpenAPIRepeatedBuildsAreIdentical checks that building the documents
// again, as StartServer does after WithOpenAPIDocs, yields the same documents,
// with const present in 3.1 and absent in 3.0 every time.
func TestOpenAPIRepeatedBuildsAreIdentical(t *testing.T) {
	o := New()
	o.Post("/things", anyHandler, DocRequestBody(&nullable31Model{}), DocResponse(&nullable31Model{}))
	o.Get("/accounts", anyHandler, DocResponse(embeddedAccount{}))
	o.Get("/examples", anyHandler, DocResponse(typedExampleModel{}))
	o.Get("/tree", anyHandler, DocResponse(recursiveNode{}))
	o.Webhook("thingCreated", http.MethodPost, DocRequestBody(&nullable31Model{}))
	o.Webhook("treeChanged", http.MethodPost, DocRequestBody(recursiveNode{}))

	var docs30, docs31 []map[string]any
	for build := 0; build < 3; build++ {
		if build < 2 {
			o.WithOpenAPIDocs(specTestConfig("Repeated builds"))
		} else {
			o.buildOpenAPISpec()
		}
		doc30 := specDocument(t, o.openapiSpec)
		doc31 := specDocument(t, o.openapiSpec31)

		status30 := jsonObject(t, documentProperties(t, doc30, "nullable31Model"), "status")
		assert.NotContains(t, status30, "const", "build %d", build)
		assert.NotContains(t, status30, extOkapiConst, "build %d", build)
		status31 := jsonObject(t, documentProperties(t, doc31, "nullable31Model"), "status")
		assert.Equal(t, "active", status31["const"], "build %d", build)
		assert.NotContains(t, status31, extOkapiConst, "build %d", build)

		forEachSpec(t, o, func(t *testing.T, spec *openapi3.T) {
			validateOpenAPIDoc(t, spec)
		})
		docs30 = append(docs30, doc30)
		docs31 = append(docs31, doc31)
	}
	for build := 1; build < len(docs30); build++ {
		assert.Equal(t, docs30[0], docs30[build], "3.0 build %d", build)
		assert.Equal(t, docs31[0], docs31[build], "3.1 build %d", build)
	}

	// The served default document carries const.
	rec := httptest.NewRecorder()
	o.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, openApiDocPath, nil))
	require.Equal(t, http.StatusOK, rec.Code)
	var served map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &served))
	assert.Equal(t, "active", jsonObject(t, documentProperties(t, served, "nullable31Model"), "status")["const"])
}

// TestOpenAPISharedSchemaRefKeepsVersions checks that a schema shared by a
// route and a webhook keeps its 3.0 form in the 3.0 document while the 3.1
// document is derived.
func TestOpenAPISharedSchemaRefKeepsVersions(t *testing.T) {
	shared := openapi3.NewSchemaRef("", &openapi3.Schema{
		Type:     &openapi3.Types{openapi3.TypeString},
		Nullable: true,
		Example:  "draft",
	})
	o := New()
	o.Post("/status", anyHandler, DocRequestBody(shared))
	o.Webhook("statusChanged", http.MethodPost, DocRequestBody(shared))

	for build := 0; build < 2; build++ {
		o.WithOpenAPIDocs(specTestConfig("Shared schema"))

		body30 := o.openapiSpec.Paths.Value("/status").Post.RequestBody.Value.Content[constJSON].Schema.Value
		require.NotNil(t, body30)
		assert.True(t, body30.Nullable, "build %d", build)
		assert.Equal(t, []string{openapi3.TypeString}, body30.Type.Slice(), "build %d", build)
		assert.Equal(t, "draft", body30.Example, "build %d", build)

		body31 := o.openapiSpec31.Paths.Value("/status").Post.RequestBody.Value.Content[constJSON].Schema.Value
		require.NotNil(t, body31)
		assert.False(t, body31.Nullable, "build %d", build)
		assert.Equal(t, []string{openapi3.TypeString, openapi3.TypeNull}, body31.Type.Slice(), "build %d", build)
		assert.Equal(t, []any{"draft"}, body31.Examples, "build %d", build)

		forEachSpec(t, o, func(t *testing.T, spec *openapi3.T) {
			validateOpenAPIDoc(t, spec)
		})
	}
}
