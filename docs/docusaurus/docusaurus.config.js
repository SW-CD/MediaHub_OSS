// @ts-check
// Note: type annotations allow type checking and IDE autocompletion

const {themes} = require('prism-react-renderer');

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'MediaHub Documentation',
  tagline: 'High-performance media acquisition and storage server',
  favicon: 'img/logo.svg',

  url: 'https://swcd.lu',
  baseUrl: '/',

  organizationName: 'swcd',
  projectName: 'MediaHub_OSS',

  onBrokenLinks: 'warn',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          path: '../manual',
          routeBasePath: '/',
          sidebarPath: require.resolve('./sidebars.js'),
        },
        blog: false,
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      }),
    ],
  ],

  themes: [
    [
      require.resolve('@easyops-cn/docusaurus-search-local'),
      {
        hashed: true,
        language: ['en'],
        docsRouteBasePath: '/',
        docsDir: '../manual',
        indexBlog: false,
      },
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      navbar: {
        title: 'MediaHub OSS',
        logo: {
          alt: 'MediaHub Logo',
          src: 'img/logo.svg',
        },
        items: [
          {
            type: 'docSidebar',
            sidebarId: 'tutorialSidebar',
            position: 'left',
            label: 'Documentation',
          },
          {
            href: 'https://github.com/denglerchr/MediaHub_OSS',
            label: 'GitHub',
            position: 'right',
          },
        ],
      },
      footer: {
        style: 'dark',
        links: [
          {
            title: 'Docs',
            items: [
              {
                label: 'Overview & Features',
                to: '/',
              },
              {
                label: 'Installation',
                to: '/installation',
              },
              {
                label: 'API Reference',
                to: '/category/api-reference',
              },
            ],
          },
          {
            title: 'Community & Source',
            items: [
              {
                label: 'GitHub Repository',
                href: 'https://github.com/denglerchr/MediaHub_OSS',
              },
              {
                label: 'Docker Hub',
                href: 'https://hub.docker.com/r/denglerchr/mediahub_oss',
              },
            ],
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} MediaHub OSS Project. Built with Docusaurus.`,
      },
      prism: {
        theme: themes.github,
        darkTheme: themes.dracula,
      },
    }),
};

module.exports = config;
