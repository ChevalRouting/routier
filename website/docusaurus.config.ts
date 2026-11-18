import {existsSync} from 'node:fs';
import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

import remarkDomains, {GITHUB_URL, REPO_URL, SITE_URL} from './src/remark/domains';

const hasReleasedVersions = existsSync('versions.json');

const GITHUB_OWNER_URL = GITHUB_URL.replace(/\/[^/]+\/?$/, '');

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

  customFields: {
    repoUrl: REPO_URL,
    githubUrl: GITHUB_URL,
  },

  organizationName: 'ChevalRouting',
  projectName: 'routier',

  onBrokenLinks: 'throw',
  onBrokenAnchors: 'throw',

  markdown: {
    hooks: {
      onBrokenMarkdownLinks: 'throw',
    },
  },

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
          exclude: ['internal/**'],
          sidebarPath: './sidebars.ts',
          beforeDefaultRemarkPlugins: [remarkDomains],
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
        {to: '/docs/quickstart', label: 'Try it', position: 'left'},
        {to: '/docs/help-test', label: 'Help test', position: 'left'},
        {href: GITHUB_URL, label: 'GitHub', position: 'right'},
        ...(hasReleasedVersions
          ? [{type: 'docsVersionDropdown', position: 'right'} as const]
          : []),
      ],
    },
    footer: {
      style: 'dark',
      copyright: [
        `<a href="${GITHUB_URL}/blob/main/LICENSE">MIT License</a>`,
        `Built by <a href="${GITHUB_OWNER_URL}">@ChevalRouting</a>`,
        buildCommit
          ? `<a href="${GITHUB_URL}/commit/${buildCommit}">${buildVersion ? `${buildVersion} (${buildCommit})` : buildCommit}</a>`
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
