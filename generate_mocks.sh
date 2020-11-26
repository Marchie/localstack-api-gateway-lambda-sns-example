#!/bin/sh

rm -r mocks/

mockgen -source=pkg/repository/redis_pooler.go -destination=mocks/pkg/repository/redis_pooler.go

AWS_SDK_GO_FILES=()
for file in "$(go env GOMODCACHE)"/github.com/aws/aws-sdk-go@*
do
  AWS_SDK_GO_FILES+=("${file}")
done

AWS_SDK_GO_CURRENT=${AWS_SDK_GO_FILES[${#AWS_SDK_GO_FILES[@]}-1]}
mockgen -source="${AWS_SDK_GO_CURRENT}/service/sns/snsiface/interface.go" -destination=mocks/third-party/aws/aws-sdk-go/service/sns/snsiface/interface.go

REDIGO_FILES=()
for file in "$(go env GOMODCACHE)"/github.com/gomodule/redigo@v1.*
do
  REDIGO_FILES+=("${file}")
done

REDIGO_CURRENT=${REDIGO_FILES[${#REDIGO_FILES[@]}-1]}
mockgen -source="${REDIGO_CURRENT}/redis/redis.go" -destination=mocks/third-party/gomodule/redigo/redis.go
