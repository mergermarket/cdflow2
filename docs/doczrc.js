export default {
  title: 'cdflow2',
  description: 'deployment tooling for continuous delivery',
  base: '/opensource/cdflow2',
  menu: [
    'Overview',
    'Installation',
    'Project Setup',
    { name: 'Commands', menu: [
      'Usage',
      'Setup',
      'Release',
      'Deploy',
      'Destroy',
      'Common Terraform Setup',
      'Shell'
    ] },
    'cdflow.yaml Reference',
    'Design'
  ],
  host: '0.0.0.0',
  src: './src'
}
