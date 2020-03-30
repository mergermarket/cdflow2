export default {
  title: 'cdflow2',
  description: 'deployment tooling for continuous delivery',
  base: '/opensource/cdflow2',
  menu: [
    'Overview',
    'Installation',
    'Project Setup',
    { name: 'Commands', menu: [
      'Setup',
      'Release',
      'Deploy',
      'Common Terraform Setup'
    ] },
    'cdflow.yaml Reference',
  ],
  host: '0.0.0.0',
  src: './src'
}  
