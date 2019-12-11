import json
import os
import sys

print('message to stdout')
print('message to stderr', file=sys.stderr)
print(json.dumps({
    "release_var_from_env": "release value from env",
}))