#!/bin/bash
set -eo pipefail
FUNCTION=$(aws cloudformation describe-stack-resource --stack-name heartpoke-email-lambda --logical-resource-id function --query 'StackResourceDetail.PhysicalResourceId' --output text)

aws lambda invoke --function-name $FUNCTION --payload file://event.json --cli-binary-format raw-in-base64-out out.json
cat out.json
echo ""
