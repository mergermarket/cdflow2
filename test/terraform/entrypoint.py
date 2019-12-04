import json
import os
import sys

# info for diagnostics
print("message to stderr", file=sys.stderr)
print(json.dumps({
    "args": sys.argv[1:], "env": dict(os.environ), "input": sys.stdin.read(),
    "cwd": os.getcwd(), "file": open('/code/mapped-dir-test').read(),
}))

with open('build-output-test', 'w') as f:
    f.write('build output')

try:
    with open('/code/source-dir-write-test', 'w') as f:
        f.write('source output')
    raise Exception('should not have been able to write to /code')
except OSError as e:
    # expected that we cannot write here
    pass