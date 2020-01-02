import json
import os
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
        assert request['ReleaseMetadata']['metadata-key'] == 'metadata-value'
        print(json.dumps({
            'Message': f'uploaded {version}' 
        }))
    elif request['Action'] == 'prepare_terraform':
        assert os.getcwd() == '/release'
        with open('/release/test', 'w') as f:
            f.write('unpacked')
        version = request['Version']
        config = request['Config']
        env = request['Env']
        print(json.dumps({
            'TerraformImage': f'terraform:image-for-{version}',
            'Env': {
                'TEST_ENV_VAR': env['TEST_ENV_VAR'],
                'TEST_CONFIG_VAR': config['TEST_CONFIG_VAR'],
            },
            'TerraformBackendType': 'a-terraform-backend-type',
            'TerraformBackendConfig': {
                'backend-config-key': 'backend-config-value',
            },
        }))
    elif request['Action'] == 'stop':
        break
    else:
        raise Exception(f'unsupported action {request["Action"]}')