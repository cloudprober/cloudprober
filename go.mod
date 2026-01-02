module github.com/cloudprober/cloudprober

go 1.24.5

require (
	cloud.google.com/go/bigquery v1.59.1
	cloud.google.com/go/compute/metadata v0.5.2
	cloud.google.com/go/logging v1.9.0
	cloud.google.com/go/pubsub v1.36.1
	github.com/Masterminds/sprig/v3 v3.3.0
	github.com/aws/aws-sdk-go-v2 v1.21.0
	github.com/aws/aws-sdk-go-v2/config v1.15.9
	github.com/aws/aws-sdk-go-v2/credentials v1.12.12
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.12.11
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.18.3
	github.com/aws/aws-sdk-go-v2/service/s3 v1.38.5
	github.com/fsnotify/fsnotify v1.7.0
	github.com/fullstorydev/grpcurl v1.9.1
	github.com/google/go-jsonnet v0.20.0
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/google/uuid v1.6.0
	github.com/hoisie/redis v0.0.0-20160730154456-b5c6e81454e0
	github.com/jackc/pgx/v5 v5.5.5
	github.com/jhump/protoreflect v1.17.0
	github.com/kylelemons/godebug v1.1.0
	github.com/miekg/dns v1.1.62
	go.opentelemetry.io/otel v1.24.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v0.44.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v0.44.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v0.44.0
	go.opentelemetry.io/otel/sdk v1.22.0
	go.opentelemetry.io/otel/sdk/metric v1.21.0
	golang.org/x/net v0.39.0
	golang.org/x/oauth2 v0.30.0
	golang.org/x/sys v0.32.0
	google.golang.org/api v0.169.0
	google.golang.org/genproto v0.0.0-20240213162025-012b6fc9bca9
	google.golang.org/grpc v1.67.1
	google.golang.org/protobuf v1.35.1
	sigs.k8s.io/yaml v1.4.0
)

require (
	cel.dev/expr v0.16.0 // indirect
	cloud.google.com/go v0.112.1 // indirect
	cloud.google.com/go/accessapproval v1.7.5 // indirect
	cloud.google.com/go/accesscontextmanager v1.8.5 // indirect
	cloud.google.com/go/aiplatform v1.60.0 // indirect
	cloud.google.com/go/analytics v0.23.0 // indirect
	cloud.google.com/go/apigateway v1.6.5 // indirect
	cloud.google.com/go/apigeeconnect v1.6.5 // indirect
	cloud.google.com/go/apigeeregistry v0.8.3 // indirect
	cloud.google.com/go/apikeys v0.6.0 // indirect
	cloud.google.com/go/appengine v1.8.5 // indirect
	cloud.google.com/go/area120 v0.8.5 // indirect
	cloud.google.com/go/artifactregistry v1.14.7 // indirect
	cloud.google.com/go/asset v1.17.2 // indirect
	cloud.google.com/go/assuredworkloads v1.11.5 // indirect
	cloud.google.com/go/automl v1.13.5 // indirect
	cloud.google.com/go/baremetalsolution v1.2.4 // indirect
	cloud.google.com/go/batch v1.8.0 // indirect
	cloud.google.com/go/beyondcorp v1.0.4 // indirect
	cloud.google.com/go/billing v1.18.2 // indirect
	cloud.google.com/go/binaryauthorization v1.8.1 // indirect
	cloud.google.com/go/certificatemanager v1.7.5 // indirect
	cloud.google.com/go/channel v1.17.5 // indirect
	cloud.google.com/go/cloudbuild v1.15.1 // indirect
	cloud.google.com/go/clouddms v1.7.4 // indirect
	cloud.google.com/go/cloudtasks v1.12.6 // indirect
	cloud.google.com/go/compute v1.25.1 // indirect
	cloud.google.com/go/contactcenterinsights v1.13.0 // indirect
	cloud.google.com/go/container v1.31.0 // indirect
	cloud.google.com/go/containeranalysis v0.11.4 // indirect
	cloud.google.com/go/datacatalog v1.19.3 // indirect
	cloud.google.com/go/dataflow v0.9.5 // indirect
	cloud.google.com/go/dataform v0.9.2 // indirect
	cloud.google.com/go/datafusion v1.7.5 // indirect
	cloud.google.com/go/datalabeling v0.8.5 // indirect
	cloud.google.com/go/dataplex v1.14.2 // indirect
	cloud.google.com/go/dataproc v1.12.0 // indirect
	cloud.google.com/go/dataproc/v2 v2.4.0 // indirect
	cloud.google.com/go/dataqna v0.8.5 // indirect
	cloud.google.com/go/datastore v1.15.0 // indirect
	cloud.google.com/go/datastream v1.10.4 // indirect
	cloud.google.com/go/deploy v1.17.1 // indirect
	cloud.google.com/go/dialogflow v1.49.0 // indirect
	cloud.google.com/go/dlp v1.11.2 // indirect
	cloud.google.com/go/documentai v1.25.0 // indirect
	cloud.google.com/go/domains v0.9.5 // indirect
	cloud.google.com/go/edgecontainer v1.1.5 // indirect
	cloud.google.com/go/errorreporting v0.3.0 // indirect
	cloud.google.com/go/essentialcontacts v1.6.6 // indirect
	cloud.google.com/go/eventarc v1.13.4 // indirect
	cloud.google.com/go/filestore v1.8.1 // indirect
	cloud.google.com/go/firestore v1.14.0 // indirect
	cloud.google.com/go/functions v1.16.0 // indirect
	cloud.google.com/go/gaming v1.10.1 // indirect
	cloud.google.com/go/gkebackup v1.3.5 // indirect
	cloud.google.com/go/gkeconnect v0.8.5 // indirect
	cloud.google.com/go/gkehub v0.14.5 // indirect
	cloud.google.com/go/gkemulticloud v1.1.1 // indirect
	cloud.google.com/go/grafeas v0.3.4 // indirect
	cloud.google.com/go/gsuiteaddons v1.6.5 // indirect
	cloud.google.com/go/iam v1.1.6 // indirect
	cloud.google.com/go/iap v1.9.4 // indirect
	cloud.google.com/go/ids v1.4.5 // indirect
	cloud.google.com/go/iot v1.7.5 // indirect
	cloud.google.com/go/kms v1.15.7 // indirect
	cloud.google.com/go/language v1.12.3 // indirect
	cloud.google.com/go/lifesciences v0.9.5 // indirect
	cloud.google.com/go/longrunning v0.5.5 // indirect
	cloud.google.com/go/managedidentities v1.6.5 // indirect
	cloud.google.com/go/maps v1.6.4 // indirect
	cloud.google.com/go/mediatranslation v0.8.5 // indirect
	cloud.google.com/go/memcache v1.10.5 // indirect
	cloud.google.com/go/metastore v1.13.4 // indirect
	cloud.google.com/go/monitoring v1.18.0 // indirect
	cloud.google.com/go/networkconnectivity v1.14.4 // indirect
	cloud.google.com/go/networkmanagement v1.9.4 // indirect
	cloud.google.com/go/networksecurity v0.9.5 // indirect
	cloud.google.com/go/notebooks v1.11.3 // indirect
	cloud.google.com/go/optimization v1.6.3 // indirect
	cloud.google.com/go/orchestration v1.8.5 // indirect
	cloud.google.com/go/orgpolicy v1.12.1 // indirect
	cloud.google.com/go/osconfig v1.12.5 // indirect
	cloud.google.com/go/oslogin v1.13.1 // indirect
	cloud.google.com/go/phishingprotection v0.8.5 // indirect
	cloud.google.com/go/policytroubleshooter v1.10.3 // indirect
	cloud.google.com/go/privatecatalog v0.9.5 // indirect
	cloud.google.com/go/pubsublite v1.8.1 // indirect
	cloud.google.com/go/recaptchaenterprise v1.3.1 // indirect
	cloud.google.com/go/recaptchaenterprise/v2 v2.9.2 // indirect
	cloud.google.com/go/recommendationengine v0.8.5 // indirect
	cloud.google.com/go/recommender v1.12.1 // indirect
	cloud.google.com/go/redis v1.14.2 // indirect
	cloud.google.com/go/resourcemanager v1.9.5 // indirect
	cloud.google.com/go/resourcesettings v1.6.5 // indirect
	cloud.google.com/go/retail v1.16.0 // indirect
	cloud.google.com/go/run v1.3.4 // indirect
	cloud.google.com/go/scheduler v1.10.6 // indirect
	cloud.google.com/go/secretmanager v1.11.5 // indirect
	cloud.google.com/go/security v1.15.5 // indirect
	cloud.google.com/go/securitycenter v1.24.4 // indirect
	cloud.google.com/go/servicecontrol v1.11.1 // indirect
	cloud.google.com/go/servicedirectory v1.11.4 // indirect
	cloud.google.com/go/servicemanagement v1.8.0 // indirect
	cloud.google.com/go/serviceusage v1.6.0 // indirect
	cloud.google.com/go/shell v1.7.5 // indirect
	cloud.google.com/go/spanner v1.56.0 // indirect
	cloud.google.com/go/speech v1.21.1 // indirect
	cloud.google.com/go/storage v1.38.0 // indirect
	cloud.google.com/go/storagetransfer v1.10.4 // indirect
	cloud.google.com/go/talent v1.6.6 // indirect
	cloud.google.com/go/texttospeech v1.7.5 // indirect
	cloud.google.com/go/tpu v1.6.5 // indirect
	cloud.google.com/go/trace v1.10.5 // indirect
	cloud.google.com/go/translate v1.10.1 // indirect
	cloud.google.com/go/video v1.20.4 // indirect
	cloud.google.com/go/videointelligence v1.11.5 // indirect
	cloud.google.com/go/vision v1.2.0 // indirect
	cloud.google.com/go/vision/v2 v2.8.0 // indirect
	cloud.google.com/go/vmmigration v1.7.5 // indirect
	cloud.google.com/go/vmwareengine v1.1.1 // indirect
	cloud.google.com/go/vpcaccess v1.7.5 // indirect
	cloud.google.com/go/webrisk v1.9.5 // indirect
	cloud.google.com/go/websecurityscanner v1.6.5 // indirect
	cloud.google.com/go/workflows v1.12.4 // indirect
	dario.cat/mergo v1.0.1 // indirect
	dmitri.shuralyov.com/gpu/mtl v0.0.0-20190408044501-666a987793e9 // indirect
	gioui.org v0.0.0-20210308172011-57750fc8a0a6 // indirect
	git.sr.ht/~sbinet/gg v0.3.1 // indirect
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/BurntSushi/xgb v0.0.0-20160522181843-27f122750802 // indirect
	github.com/JohnCGriffin/overflow v0.0.0-20211019200055-46fa312c352c // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.3.0 // indirect
	github.com/OneOfOne/xxhash v1.2.2 // indirect
	github.com/ajstarks/deck v0.0.0-20200831202436-30c9fc6549a9 // indirect
	github.com/ajstarks/deck/generate v0.0.0-20210309230005-c3f852c02e19 // indirect
	github.com/ajstarks/svgo v0.0.0-20211024235047-1546f124cd8b // indirect
	github.com/alecthomas/assert/v2 v2.3.0 // indirect
	github.com/alecthomas/participle/v2 v2.1.0 // indirect
	github.com/alecthomas/repr v0.2.0 // indirect
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/antihax/optional v1.0.0 // indirect
	github.com/apache/arrow/go/v10 v10.0.1 // indirect
	github.com/apache/arrow/go/v11 v11.0.0 // indirect
	github.com/apache/arrow/go/v12 v12.0.1 // indirect
	github.com/apache/arrow/go/v14 v14.0.2 // indirect
	github.com/apache/thrift v0.17.0 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.4.13 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.41 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.35 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.18 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.1.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.9.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.1.36 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.35 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.15.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.11.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.16.12 // indirect
	github.com/aws/smithy-go v1.14.2 // indirect
	github.com/boombuler/barcode v1.0.1 // indirect
	github.com/bufbuild/protocompile v0.14.1 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/chzyer/logex v1.2.0 // indirect
	github.com/chzyer/readline v1.5.0 // indirect
	github.com/chzyer/test v0.0.0-20210722231415-061457976a23 // indirect
	github.com/client9/misspell v0.3.4 // indirect
	github.com/cncf/udpa/go v0.0.0-20220112060539-c52dc94e7fbe // indirect
	github.com/cncf/xds/go v0.0.0-20240723142845-024c85f92f20 // indirect
	github.com/creack/pty v1.1.9 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docopt/docopt-go v0.0.0-20180111231733-ee0de3bc6815 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fatih/color v1.15.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fogleman/gg v1.3.0 // indirect
	github.com/frankban/quicktest v1.14.6 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-fonts/dejavu v0.1.0 // indirect
	github.com/go-fonts/latin-modern v0.2.0 // indirect
	github.com/go-fonts/liberation v0.2.0 // indirect
	github.com/go-fonts/stix v0.1.0 // indirect
	github.com/go-gl/glfw v0.0.0-20190409004039-e6da0acd62b1 // indirect
	github.com/go-gl/glfw/v3.3/glfw v0.0.0-20200222043503-6f7a984d4dc4 // indirect
	github.com/go-latex/latex v0.0.0-20210823091927-c0d11ff05a81 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-pdf/fpdf v0.6.0 // indirect
	github.com/go-playground/assert/v2 v2.0.1 // indirect
	github.com/go-playground/locales v0.13.0 // indirect
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/go-playground/validator/v10 v10.4.1 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/goccy/go-yaml v1.11.0 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/golang/glog v1.2.2 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/google/flatbuffers v23.5.26+incompatible // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/go-pkcs11 v0.2.1-0.20230907215043-c6f79328ddf9 // indirect
	github.com/google/gofuzz v1.0.0 // indirect
	github.com/google/martian v2.1.0+incompatible // indirect
	github.com/google/martian/v3 v3.3.2 // indirect
	github.com/google/pprof v0.0.0-20221118152302-e6195bd50e26 // indirect
	github.com/google/renameio v0.1.0 // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/go-type-adapters v1.0.0 // indirect
	github.com/googleapis/google-cloud-go-testing v0.0.0-20200911160855-bcd43fbb19e8 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.16.0 // indirect
	github.com/hashicorp/golang-lru v0.5.1 // indirect
	github.com/hexops/gotextdiff v1.0.3 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/iancoleman/strcase v0.3.0 // indirect
	github.com/ianlancetaylor/demangle v0.0.0-20220319035150-800ac71e25c2 // indirect
	github.com/itchyny/timefmt-go v0.1.4 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/jhump/gopoet v0.1.0 // indirect
	github.com/jhump/goprotoc v0.5.0 // indirect
	github.com/jmespath/go-jmespath/internal/testify v1.5.1 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/jstemmer/go-junit-report v0.9.1 // indirect
	github.com/jung-kurt/gofpdf v1.0.3-0.20190309125859-24315acbbda5 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kisielk/gotool v1.0.0 // indirect
	github.com/klauspost/asmfmt v1.3.2 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/klauspost/cpuid/v2 v2.2.5 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/pty v1.1.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/lyft/protoc-gen-star v0.6.1 // indirect
	github.com/lyft/protoc-gen-star/v2 v2.0.4-0.20230330145011-496ad1ac90a4 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/mattn/go-sqlite3 v1.14.16 // indirect
	github.com/minio/asm2plan9s v0.0.0-20200509001527-cdd76441f9d8 // indirect
	github.com/minio/c2goasm v0.0.0-20190812172519-36a3d3bbc4f3 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180228061459-e0a39a4cb421 // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/phpdave11/gofpdf v1.4.2 // indirect
	github.com/phpdave11/gofpdi v1.0.13 // indirect
	github.com/pierrec/lz4/v4 v4.1.18 // indirect
	github.com/pkg/diff v0.0.0-20210226163009-20ebb0f2a09e // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pkg/sftp v1.13.1 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.6.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/rogpeppe/fastuuid v1.2.0 // indirect
	github.com/rogpeppe/go-internal v1.11.0 // indirect
	github.com/ruudk/golang-pdf417 v0.0.0-20201230142125-a7e3863a1245 // indirect
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/spaolacci/murmur3 v0.0.0-20180118202830-f09979ecbc72 // indirect
	github.com/spf13/afero v1.10.0 // indirect
	github.com/spf13/cast v1.7.0 // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/substrait-io/substrait-go v0.4.2 // indirect
	github.com/yuin/goldmark v1.4.13 // indirect
	github.com/zeebo/assert v1.3.0 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	go.einride.tech/aip v0.66.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.49.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.49.0 // indirect
	go.opentelemetry.io/otel/metric v1.24.0 // indirect
	go.opentelemetry.io/otel/trace v1.24.0 // indirect
	go.opentelemetry.io/proto/otlp v1.0.0 // indirect
	go.uber.org/goleak v1.3.0 // indirect
	golang.org/x/exp v0.0.0-20231006140011-7918f672742d // indirect
	golang.org/x/image v0.0.0-20220302094943-723b81ca9867 // indirect
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616 // indirect
	golang.org/x/mobile v0.0.0-20190719004257-d2bd2a29d028 // indirect
	golang.org/x/telemetry v0.0.0-20240521205824-bda55230c457 // indirect
	golang.org/x/term v0.31.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	gonum.org/v1/gonum v0.12.0 // indirect
	gonum.org/v1/netlib v0.0.0-20190313105609-8cb42192e0e0 // indirect
	gonum.org/v1/plot v0.10.1 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240814211410-ddb44dafa142 // indirect
	google.golang.org/genproto/googleapis/bytestream v0.0.0-20240304161311-37d4d3c04a78 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240814211410-ddb44dafa142 // indirect
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.1.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/errgo.v2 v2.1.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gotest.tools/v3 v3.5.1 // indirect
	honnef.co/go/tools v0.1.3 // indirect
	lukechampine.com/uint128 v1.3.0 // indirect
	modernc.org/cc/v3 v3.40.0 // indirect
	modernc.org/ccgo/v3 v3.16.13 // indirect
	modernc.org/ccorpus v1.11.6 // indirect
	modernc.org/httpfs v1.0.6 // indirect
	modernc.org/libc v1.22.4 // indirect
	modernc.org/mathutil v1.5.0 // indirect
	modernc.org/memory v1.5.0 // indirect
	modernc.org/opt v0.1.3 // indirect
	modernc.org/sqlite v1.21.2 // indirect
	modernc.org/strutil v1.1.3 // indirect
	modernc.org/tcl v1.15.1 // indirect
	modernc.org/token v1.1.0 // indirect
	modernc.org/z v1.7.0 // indirect
	rsc.io/binaryregexp v0.2.0 // indirect
	rsc.io/pdf v0.1.1 // indirect
	rsc.io/quote/v3 v3.1.0 // indirect
	rsc.io/sampler v1.3.0 // indirect
)

require (
	github.com/census-instrumentation/opencensus-proto v0.4.1 // indirect
	github.com/envoyproxy/go-control-plane v0.13.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.1.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/googleapis/gax-go/v2 v2.12.2 // indirect
	github.com/itchyny/gojq v0.12.9
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/stretchr/testify v1.9.0
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/crypto v0.37.0 // indirect
	golang.org/x/mod v0.18.0 // indirect
	golang.org/x/sync v0.13.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	golang.org/x/tools v0.22.0
)
