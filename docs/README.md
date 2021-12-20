Documentation here is published to:

https://developer.ionanalytics.com/opensource/cdflow2

Publishing is done by this internal ION Analytics repo/pipeline:

https://github.com/ion-analytics/open-source-docs

To preview changes you have locally, clone that repo next to `cdflow2` and run the following
from the root of the repo:

```
# install dependencies
npm ci

# import from your local clone
./scripts/import-docs-local.sh cdflow2 ../cdflow2

# run the site locally to preview (http://localhost:3000/opensource/cdflow2)
npm run dev
```