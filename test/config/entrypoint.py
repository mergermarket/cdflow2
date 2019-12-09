import json
import sys

for line in sys.stdin:
    request = json.loads(line)
    if request['Action'] == 'configure_release':
        print(json.dumps({
            'Env': {
                'TEST_RELEASE_VAR_FROM_ENV': request['Env']['TEST_ENV_VAR'],
                'TEST_RELEASE_VAR_FROM_CONFIG': request['Config']['TEST_CONFIG_VAR']
            }
        }))
    elif request['Action'] == 'stop':
        break
    else:
        raise Exception(f'unsupported action {request["Action"]}')