import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  docs: [
    'README',
    'architecture',
    'installation',
    'configuration',
    {
      type: 'category',
      label: 'Networking',
      items: ['interfaces', 'routing', 'wireguard', 'firewall', 'dhcp', 'dns', 'tools'],
    },
    {
      type: 'category',
      label: 'Features',
      items: ['system', 'friends', 'monitoring', 'macros'],
    },
    {
      type: 'category',
      label: 'Reference',
      items: ['cli', 'api', 'building', 'testing', 'code-style'],
    },
  ],
};

export default sidebars;
