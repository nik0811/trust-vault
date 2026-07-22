const withNextra = require('nextra')({
  theme: 'nextra-theme-docs',
  themeConfig: './theme.config.tsx',
  defaultShowCopyCode: true,
})

module.exports = withNextra({
  basePath: '/docs',
  images: {
    unoptimized: true,
  },
  output: 'export',
  trailingSlash: true,
})
