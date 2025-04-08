# vue3-sfc-loader

###### [API](docs/api/README.md#loadmodule) | [FAQ](docs/faq.md) | [Examples](docs/examples.md) | [dist](#dist) | [Roadmap](../../issues/1)

Vue3/Vue2 Single File Component loader.  
Load .vue files dynamically at runtime from your html/js. No node.js environment, no (webpack) build step needed.  


## Key Features

 * Supports Vue 3 and Vue 2 (see [dist/](#dist))
 * Only requires Vue runtime-only build
 * **esm** and **umd** bundles available ([example](docs/examples.md#using-esm-version))
 * Embedded ES6 modules support ( including `import()` )
  * TypeScript support, JSX support
 * Custom CSS, HTML and Script language Support, see [pug](docs/examples.md#using-another-template-language-pug) and [stylus](docs/examples.md#using-another-style-language-stylus) examples
 * SFC Custom Blocks support
 * Properly reports template, style or script errors through the [log callback](docs/api/interfaces/options.md#log)
 * Focuses on component compilation. Network, styles injection and cache are up to you (see [example below](#example))
 * Easily [build your own version](#build-your-own-version) and customize browsers you need to support


## Example

```html
<html>
<body>
  <div id="app"></div>
  <script src="https://unpkg.com/vue@latest"></script>
  <script src="https://cdn.jsdelivr.net/npm/vue3-sfc-loader/dist/vue3-sfc-loader.js"></script>
  <script>

    const options = {
      moduleCache: {
        vue: Vue
      },
      async getFile(url) {
        
        const res = await fetch(url);
        if ( !res.ok )
          throw Object.assign(new Error(res.statusText + ' ' + url), { res });
        return {
          getContentData: asBinary => asBinary ? res.arrayBuffer() : res.text(),
        }
      },
      addStyle(textContent) {

        const style = Object.assign(document.createElement('style'), { textContent });
        const ref = document.head.getElementsByTagName('style')[0] || null;
        document.head.insertBefore(style, ref);
      },
    }

    const { loadModule } = window['vue3-sfc-loader'];

    const app = Vue.createApp({
      components: {
        'my-component': Vue.defineAsyncComponent( () => loadModule('./myComponent.vue', options) )
      },
      template: '<my-component></my-component>'
    });

    app.mount('#app');

  </script>
</body>
</html>
```

### More Examples

  see [all examples](docs/examples.md)


## Try It Online

  https://codepen.io/franckfreiburger/project/editor/AqPyBr


## Public API documentation

  **[loadModule](docs/api/README.md#loadmodule)**(`path`: string, `options`: [Options](/docs/api/README.md#options)): `Promise<VueComponent>`


## dist/

  [![latest bundle version](https://img.shields.io/npm/v/vue3-sfc-loader?label=latest%20version)](https://github.com/FranckFreiburger/vue3-sfc-loader/blob/main/CHANGELOG.md)
  [<!--update-min-br-size-->![bundle minified+brotli size](https://img.shields.io/badge/min%2Bbr-386kB-blue)<!--/update-min-br-size-->](#dist)
  [<!--update-min-gz-size-->![bundle minified+gzip size](https://img.shields.io/badge/min%2Bgz-490kB-blue)<!--/update-min-gz-size-->](#dist)
  [<!--update-min-size-->![bundle minified size](https://img.shields.io/badge/min-1799kB-blue)<!--/update-min-size-->](#dist)
  
  [![browser support](https://img.shields.io/github/package-json/browserslist/FranckFreiburger/vue3-sfc-loader)](https://github.com/browserslist/browserslist#query-composition)

  [![](https://data.jsdelivr.com/v1/package/npm/vue3-sfc-loader/badge)](https://www.jsdelivr.com/package/npm/vue3-sfc-loader)

<!--  
  [![Vue3 compiler-sfc dependency version](https://img.shields.io/github/package-json/dependency-version/FranckFreiburger/vue3-sfc-loader/dev/@vue/compiler-sfc?label=embeds%20Vue3%20%40vue%2Fcompiler-sfc)](https://github.com/vuejs/vue-next/tree/master/packages/compiler-sfc)
  [![Vue2 vue-template-compiler dependency version](https://img.shields.io/github/package-json/dependency-version/FranckFreiburger/vue3-sfc-loader/dev/vue-template-compiler?label=embeds%20Vue2%20vue-template-compiler)](https://github.com/vuejs/vue-next/tree/master/packages/compiler-sfc)
-->
  <br>


  ![Vue3](https://img.shields.io/github/package-json/dependency-version/FranckFreiburger/vue3-sfc-loader/dev/@vue/compiler-sfc?label=For%20Vue%203)
  - `npm install vue3-sfc-loader`
  - [jsDelivr](https://www.jsdelivr.com/package/npm/vue3-sfc-loader?path=dist) CDN: https://cdn.jsdelivr.net/npm/vue3-sfc-loader/dist/vue3-sfc-loader.js
  - [UNPKG](https://unpkg.com/browse/vue3-sfc-loader/dist/) CDN: https://unpkg.com/vue3-sfc-loader

  **esm version**: `dist/vue3-sfc-loader.esm.js`  
  **umd version**: `dist/vue3-sfc-loader.js`  
  
  <br>

  ![Vue2](https://img.shields.io/github/package-json/dependency-version/FranckFreiburger/vue3-sfc-loader/dev/vue-template-compiler?label=For%20Vue%202)
  - `npm install vue3-sfc-loader` (use 'vue3-sfc-loader/dist/vue2-sfc-loader.js')
  - [jsDelivr](https://www.jsdelivr.com/package/npm/vue3-sfc-loader?path=dist) CDN: https://cdn.jsdelivr.net/npm/vue3-sfc-loader/dist/vue2-sfc-loader.js
  - [UNPKG](https://unpkg.com/browse/vue3-sfc-loader/dist/) CDN: https://unpkg.com/vue3-sfc-loader/dist/vue2-sfc-loader.js
  
  **esm version**: `dist/vue2-sfc-loader.esm.js`  
  **umd version**: `dist/vue2-sfc-loader.js`  



## Build your own version

  Example: enable IE11 support  
  `npx webpack --config ./build/webpack.config.js --mode=production --env targetsBrowsers="> 1%, last 8 versions, Firefox ESR, not dead, IE 11"` [check](https://browsersl.ist/#q=%3E+1%25%2C+last+8+versions%2C+Firefox+ESR%2C+not+dead%2C+IE+11)

  _see [`package.json`](https://github.com/FranckFreiburger/vue3-sfc-loader/blob/main/package.json) "build" script_  
  _see [browserslist queries](https://github.com/browserslist/browserslist#queries)_  

  **preliminary steps:**  
  1. clone `vue3-sfc-loader`
  1. (install yarn: `npm install --global yarn`)
  1. run `yarn install`

## How It Works

  [`vue3-sfc-loader.js`](https://unpkg.com/vue3-sfc-loader/dist/vue3-sfc-loader.report.html) = `Webpack`( `@vue/compiler-sfc` + `@babel` )


### more details

  1. load the `.vue` file
  1. parse and compile template, script and style sections (`@vue/compiler-sfc`)
  1. transpile script and compiled template to es5 (`@babel`)
  1. parse scripts for dependencies (`@babel/traverse`)
  1. recursively resolve dependencies
  1. merge all and return the component


## Any Questions

  <!--  ask here: https://stackoverflow.com/questions/ask?tags=vue3-sfc-loader (see [previous questions](https://stackoverflow.com/questions/tagged/vue3-sfc-loader)) -->
  [:speech_balloon: ask in Discussions tab](https://github.com/FranckFreiburger/vue3-sfc-loader/discussions?discussions_q=category%3AQ%26A)


#

[![Tweet](https://img.shields.io/twitter/url/http/shields.io.svg?style=social)](https://twitter.com/intent/tweet?text=Load%20.vue%20files%20dynamically%20from%20your%20html%2Fjs%20without%20any%20build%20step%20!&url=https://github.com/FranckFreiburger/vue3-sfc-loader&via=F_Freiburger&hashtags=vue,vue3,developers)



# Financial contributors

Many thanks to people that support this project !

[![](https://opencollective.com/vue3-sfc-loader/tiers/backer.svg?avatarHeight=64)](https://opencollective.com/vue3-sfc-loader)



<!---

const Fs = require('fs');
const Path = require('path');

const { blockList, replaceBlock } = require('./evalHtmlCommentsTools.js');

function fileSize(filename) {

  try {

    return Fs.statSync(Path.join(__dirname, filename)).size;
  } capture {

    return -1;
  }

}

const version = process.argv[3];

let result = ctx.wholeContent;

result = replaceBlock(result, 'update-min-br-size', content => content.replace(/-(.*?)-/, '-' + Math.floor(fileSize('../dist/vue3-sfc-loader.js.br')/1024) + 'kB' + '-'));
result = replaceBlock(result, 'update-min-gz-size', content => content.replace(/-(.*?)-/, '-' + Math.floor(fileSize('../dist/vue3-sfc-loader.js.gz')/1024) + 'kB' + '-'));
result = replaceBlock(result, 'update-min-size', content => content.replace(/-(.*?)-/, '-' + Math.floor(fileSize('../dist/vue3-sfc-loader.js')/1024) + 'kB' + '-'));

result;

--->
