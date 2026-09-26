import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'Okapi',
  tagline:
    'A lightweight, fast, modular HTTP framework for Go with automatic request validation and built-in OpenAPI documentation',
  favicon: 'img/favicon.ico',

  future: {
    v4: true,
  },

  url: 'https://okapi.jkaninda.dev',
  baseUrl: '/',

  organizationName: 'jkaninda',
  projectName: 'okapi',

  onBrokenLinks: 'throw',
  onBrokenAnchors: 'warn',

  // The Jekyll site used pretty permalinks, so every published URL ended in a
  // slash. Keeping that shape means the links already out in the world resolve
  // without a redirect.
  trailingSlash: true,

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  markdown: {
    mermaid: true,
    // 'detect' parses .md as CommonMark and only .mdx as MDX. The pages came
    // from Jekyll and are full of Go generics and struct tags, which MDX would
    // read as JSX.
    format: 'detect',
  },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          editUrl: 'https://github.com/jkaninda/okapi/tree/main/docs/',
          // Docs are served from the site root, as the Jekyll site served them.
          routeBasePath: '/',
          showLastUpdateTime: true,
          // v1.0.0 is what visitors get at the root; v0.11.0 stays browsable at
          // /0.11.0/ for applications that have not upgraded yet.
          lastVersion: 'current',
          versions: {
            current: {
              label: 'v1.0.0',
              path: '',
              badge: false,
            },
            '0.11.0': {
              label: 'v0.11.0',
              path: '0.11.0',
              banner: 'unmaintained',
            },
          },
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
        sitemap: {
          changefreq: 'weekly',
          priority: 0.5,
        },
      } satisfies Preset.Options,
    ],
  ],

  plugins: [
    [
      '@docusaurus/plugin-client-redirects',
      {
        // The Jekyll site published pages with a `.html` suffix. Docusaurus
        // serves them extensionless, so map the old form onto the new one and
        // keep the accumulated inbound links working.
        createRedirects(existingPath: string) {
          const path = existingPath.replace(/\/$/, '');
          // Skip the site root and pages that are already a `.html` file,
          // which would otherwise redirect from `404.html.html`.
          if (path === '' || path.endsWith('.html')) {
            return undefined;
          }
          return [`${path}.html`];
        },
        redirects: [
          {
            // The page was published for a year with the word misspelled.
            to: '/features/error-handling',
            from: ['/features/error-hanling'],
          },
        ],
      },
    ],
  ],

  themes: [
    '@docusaurus/theme-mermaid',
    [
      '@easyops-cn/docusaurus-search-local',
      {
        hashed: true, // cache-bust the index when content changes
        indexBlog: false, // blog is disabled above
        docsRouteBasePath: '/',
        highlightSearchTermsOnTargetPage: true,
        searchResultLimits: 10,
        explicitSearchResultPath: true,
      },
    ],
  ],

  themeConfig: {
    colorMode: {
      defaultMode: 'light',
      respectPrefersColorScheme: true,
    },
    navbar: {
      title: 'Okapi',
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'docsSidebar',
          position: 'left',
          label: 'Documentation',
        },
        {
          type: 'docsVersionDropdown',
          position: 'right',
          dropdownActiveClassDisabled: true,
        },
        {
          href: 'https://github.com/jkaninda/okapi/tree/main/examples',
          label: 'Examples',
          position: 'right',
        },
        {
          href: 'https://github.com/jkaninda/okapi',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Documentation',
          items: [
            {label: 'Overview', to: '/'},
            {label: 'Installation', to: '/installation/'},
            {label: 'Quickstart', to: '/quickstart/'},
            {label: 'Why Okapi', to: '/why-use-okapi/'},
          ],
        },
        {
          title: 'Reference',
          items: [
            {label: 'Core Concepts', to: '/core-concepts/'},
            {label: 'Features', to: '/features/'},
            {label: 'Validation', to: '/features/validation/'},
            {label: 'OpenAPI', to: '/features/openapi/'},
          ],
        },
        {
          title: 'Project',
          items: [
            {label: 'GitHub', href: 'https://github.com/jkaninda/okapi'},
            {
              label: 'Examples',
              href: 'https://github.com/jkaninda/okapi/tree/main/examples',
            },
            {
              label: 'Basic example',
              href: 'https://github.com/jkaninda/okapi-example',
            },
            {
              label: 'Report an issue',
              href: 'https://github.com/jkaninda/okapi/issues',
            },
          ],
        },
      ],
      copyright: `Copyright © 2025-${new Date().getFullYear()} <a href="https://www.jkaninda.dev" target="_blank" rel="noopener">Jonas Kaninda</a>. Distributed under the MIT License.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['bash', 'json', 'go', 'yaml', 'toml', 'docker'],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
