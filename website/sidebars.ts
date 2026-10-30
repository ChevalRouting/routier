import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  docs: [
    {
      type: 'category',
      label: 'Getting Started',
      collapsible: false,
      items: [{type: 'doc', id: 'README', label: 'Introduction'}, 'quickstart', 'installation', 'help-test'],
    },
    {
      type: 'category',
      label: 'Concepts',
      items: ['configuration', 'architecture'],
    },
    {
      type: 'category',
      label: 'Networking',
      items: ['interfaces', 'routing', 'wireguard', 'firewall', 'dhcp', 'dns', 'ha', 'switches'],
    },
    {
      type: 'category',
      label: 'System',
      items: ['system', 'web-tls', 'friends', 'monitoring', 'macros', 'tools'],
    },
    {
      type: 'category',
      label: 'Reference',
      items: ['cli', 'api', 'mcp'],
    },
    {
      type: 'category',
      label: 'Development',
      items: ['building', 'testing', 'code-style', 'ui-guidelines'],
    },
  ],
};

export default sidebars;
