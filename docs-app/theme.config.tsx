import React from 'react'
import { DocsThemeConfig } from 'nextra-theme-docs'

const config: DocsThemeConfig = {
  logo: (
    <span style={{ fontWeight: 700, fontSize: '1.2rem' }}>
      SecureLens
    </span>
  ),
  project: {
    link: 'https://github.com/securelens/securelens',
  },
  docsRepositoryBase: 'https://github.com/securelens/securelens/tree/main/docs-app',
  footer: {
    text: `© ${new Date().getFullYear()} SecureLens. All rights reserved.`,
  },
  head: (
    <>
      <meta name="viewport" content="width=device-width, initial-scale=1.0" />
      <meta property="og:title" content="SecureLens Documentation" />
      <meta property="og:description" content="Enterprise Data & AI Trust Platform" />
      <link rel="icon" href="/favicon.ico" />
    </>
  ),
  useNextSeoProps() {
    return {
      titleTemplate: '%s – SecureLens Docs'
    }
  },
  sidebar: {
    defaultMenuCollapseLevel: 1,
    toggleButton: true,
  },
  toc: {
    backToTop: true,
  },
  primaryHue: 210,
  banner: {
    key: 'securelens-1.0',
    text: (
      <a href="/getting-started/quickstart" target="_blank">
        SecureLens 1.0 is here! Get started →
      </a>
    ),
  },
  navigation: {
    prev: true,
    next: true,
  },
  editLink: {
    text: 'Edit this page on GitHub →',
  },
  feedback: {
    content: 'Question? Give us feedback →',
    labels: 'feedback',
  },
}

export default config
