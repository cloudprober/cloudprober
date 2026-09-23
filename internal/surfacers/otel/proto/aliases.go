// Copyright 2026 The Cloudprober Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proto

// The OTLP exporter messages used to be defined in this package. They now live
// in internal/otel/proto, shared with the tracing config. These aliases keep
// existing Go callers working, and keep them in the generated
// pkg/protos/surfacers facade (as OtelHTTPExporter, OtelGRPCExporter, etc.)
// under their original names.

import otelpb "github.com/cloudprober/cloudprober/internal/otel/proto"

type Compression = otelpb.Compression
type HTTPExporter = otelpb.HTTPExporter
type GRPCExporter = otelpb.GRPCExporter

const (
	Compression_NONE = otelpb.Compression_NONE
	Compression_GZIP = otelpb.Compression_GZIP
)
