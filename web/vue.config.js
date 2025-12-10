// web/vue.config.js
const { defineConfig } = require('@vue/cli-service')

module.exports = defineConfig({
  transpileDependencies: true,

  // Configure for single-spa MFE
  configureWebpack: {
    output: {
      // Output as SystemJS module for single-spa
      libraryTarget: 'system',
      // Unique name for the MFE
      library: {
        type: 'system',
      },
    },
    externals: [
      // Externalize shared dependencies (loaded by shell)
      'vue',
      'vue-router',
      'pinia',
      /^@platform\/.+/,
    ],
  },

  // Disable chunk splitting for single-spa
  chainWebpack: (config) => {
    config.optimization.delete('splitChunks')

    // Disable HTML plugin for library mode
    config.plugins.delete('html')
    config.plugins.delete('preload')
    config.plugins.delete('prefetch')
  },

  // Dev server config
  devServer: {
    port: 3000,
    headers: {
      'Access-Control-Allow-Origin': '*',
    },
  },

  // CSS configuration
  css: {
    loaderOptions: {
      sass: {
        additionalData: `@import "@/styles/_variables.scss";`,
      },
    },
  },
})

