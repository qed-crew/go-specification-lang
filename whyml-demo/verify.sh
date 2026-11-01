#!/bin/sh
set -eu

repo_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
why3_bin=${WHY3:-why3}
cd "$repo_dir/framework"

go generate ./...
go test -count=1 ./plugin/whyml ./cmd/go2whyml
(cd ../whyml-demo && go test -count=1 -v ./...)

mkdir -p ../whyml-demo/generated
if ! WHY3="$why3_bin" go test -count=1 -tags=integration -v ./plugin/whyml \
    > ../whyml-demo/generated/integration.log; then
    cat ../whyml-demo/generated/integration.log
    exit 1
fi
cat ../whyml-demo/generated/integration.log
for example in insertion_sort semantics; do
    go run ./cmd/go2whyml -int-bits 64 \
        -o "../whyml-demo/generated/$example.mlw" "../whyml-demo/$example.go"
    set --
    if [ -n "${WHY3_CONFIG:-}" ]; then
        set -- -C "$WHY3_CONFIG"
    fi
    if ! "$why3_bin" "$@" prove "../whyml-demo/generated/$example.mlw" \
        -a split_vc -P z3 -t 10 > "../whyml-demo/generated/$example.proof.log"; then
        cat "../whyml-demo/generated/$example.proof.log"
        exit 1
    fi
    cat "../whyml-demo/generated/$example.proof.log"
done
