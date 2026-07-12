import {existsSync} from 'node:fs';
import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

import remarkDomains, {SITE_URL} from './src/remark/domains';

const hasReleasedVersions = existsSync('versions.json');

const buildCommit = process.env.SITE_COMMIT;
const buildVersion = process.env.SITE_VERSION ?? '';

const config: Config = {
  title: 'Routier',
  tagline: 'Declarative router and firewall appliance',
  favicon: 'img/logo.svg',

  future: {
    v4: true,
  },

  url: SITE_URL,
  baseUrl: process.env.SITE_BASE_URL ?? '/',

  organizationName: 'ChevalRouting',
  projectName: 'routier',

  onBrokenLinks: 'throw',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          path: '../docs',
          routeBasePath: '/docs',
          sidebarPath: './sidebars.ts',
          remarkPlugins: [remarkDomains],
          includeCurrentVersion: !hasReleasedVersions,
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    colorMode: {
      respectPrefersColorScheme: true,
    },
    navbar: {
      title: 'Routier',
      logo: {
        alt: 'Routier',
        src: 'img/logo.svg',
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'docs',
          position: 'left',
          label: 'Documentation',
        },
        ...(hasReleasedVersions
          ? [{type: 'docsVersionDropdown', position: 'right'} as const]
          : []),
      ],
    },
    footer: {
      style: 'dark',
      copyright: [
        '<a href="https://github.com/ChevalRouting/routier/blob/main/LICENSE">MIT License</a>',
        'Built by <a href="https://github.com/ChevalRouting">@ChevalRouting</a>',
        buildCommit
          ? `<a href="https://github.com/ChevalRouting/routier/commit/${buildCommit}">${buildVersion ? `${buildVersion} (${buildCommit})` : buildCommit}</a>`
          : null,
      ]
        .filter(Boolean)
        .join(' &middot; '),
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['yaml', 'bash', 'json'],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
