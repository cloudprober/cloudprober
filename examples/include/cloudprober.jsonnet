# Note that:
#  - `include` should be the first word on the line.
#  - `include` path should be relative to the directory of the file including
#    it.
#  - `include` supports glob patterns, so you can include multiple files with
#    one directive, e.g. `include "cloudprober.d/*.cfg"`.
#  - `include` does a simple text replacement, without any thought to the
#    actual content.
#
# See https://github.com/cloudprober/cloudprober/tree/main/config/testdata/include_test 
# for more include examples.
include "teams/team1.jsonnet"
include "teams/team2.jsonnet"

{ probe: team1_probes + team2_probes }