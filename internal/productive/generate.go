// Package productive is a hand-written JSON:API client for the Productive.io
// time-tracking API, built on top of oapi-codegen-generated resource models.
//
// Productive's OpenAPI response/request bodies reference resource
// properties via deep $refs (e.g.
// #/components/schemas/resource_booking/properties/approved) that
// oapi-codegen's operation/response codegen can't follow, so only the plain
// resource models (spec/models-only.yaml) are generated; the JSON:API
// envelope (data/attributes/meta) and HTTP calls are hand-written in this
// package. See spec/tools/update_spec.py for how the spec files are derived.
package productive

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 -config oapi-codegen.yaml spec/models-only.yaml
