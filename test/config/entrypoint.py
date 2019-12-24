import json
import sys

version = None

for line in sys.stdin:
    request = json.loads(line)
    if request['Action'] == 'configure_release':
        version = request['Version']
        print(json.dumps({
            'Env': {
                'TEST_VERSION': version,
                'TEST_RELEASE_VAR_FROM_ENV': request['Env']['TEST_ENV_VAR'],
                'TEST_RELEASE_VAR_FROM_CONFIG': request['Config']['TEST_CONFIG_VAR']
            }
        }))
    elif request['Action'] == 'upload_release':
        print(json.dumps({}))
    elif request['Action'] == 'prepare_terraform':
        version = request['Version']
        print(json.dumps({
            'TerraformImage': f'terraform:image-for-{version}',
            'Env': {
                'EnvKey': 'EnvValue',
            },
        }))
    elif request['Action'] == 'stop':
        break
    else:
        raise Exception(f'unsupported action {request["Action"]}')