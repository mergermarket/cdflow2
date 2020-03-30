#!/bin/bash

set -e

if [ "$1" == "live" ]; then
    fastly_site_id=0D4hN8fWNcMnEp6CbAIDup
    bucket=live-developer-frontend-router-static-content
elif [ "$2" == "aslive" ]; then
    fastly_site_id=08pLXpbfh088YBYI1c4Qlk
    bucket=aslive-developer-frontend-router-static-content
else
    echo "environment parameter is required - aslive or live" >&2
    exit 1
fi

aws s3 sync --delete --metadata 'Surrogate-Key=cdflow2' docs/dist/ s3://$bucket/opensource/cdflow2/

if [ "$FASTLY_API_KEY" != "" ]; then
    curl --fail \
        --show-error \
        --silent \
        --request POST \
        --header 'Accept: application/json' \
        --header "Fastly-Key: $FASTLY_API_KEY" \
        https://api.fastly.com/service/$fastly_site_id/purge/cdflow2
fi
