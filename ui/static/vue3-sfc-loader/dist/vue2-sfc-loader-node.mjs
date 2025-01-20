import { posix } from "path";

import { transformFromAstAsync, traverse, types } from "@babel/core";

import { parse as parse$1 } from "@babel/parser";

import { codeFrameColumns } from "@babel/code-frame";

import babelPluginTransformModulesCommonjs from "@babel/plugin-transform-modules-commonjs";

import babelPlugin_typescript from "@babel/plugin-transform-typescript";

import SparkMD5 from "spark-md5";

import { parse, compileTemplate, compileStyleAsync } from "@vue/component-compiler-utils";

import * as vueTemplateCompiler from "vue-template-compiler";

import jsx from "@vue/babel-plugin-transform-vue-jsx";

import babelSugarInjectH from "@vue/babel-sugar-inject-h";

const merge = require("merge-source-map");

const customRequire = (name, options) => {
    var _a;
    const requireFn = (_a = options === null || options === void 0 ? void 0 : options.preprocessOptions) === null || _a === void 0 ? void 0 : _a.customRequire;
    if (requireFn) {
        return requireFn(name);
    }
    return require(name);
};

const scss = {
    render(source, map, options) {
        const nodeSass = customRequire("sass", options);
        const finalOptions = Object.assign({}, options, {
            data: source,
            file: options.filename,
            outFile: options.filename,
            sourceMap: !!map
        });
        try {
            const result = nodeSass.renderSync(finalOptions);
            if (map) {
                return {
                    code: result.css.toString(),
                    map: merge(map, JSON.parse(result.map.toString())),
                    errors: []
                };
            }
            return {
                code: result.css.toString(),
                errors: []
            };
        } catch (e) {
            return {
                code: "",
                errors: [ e ]
            };
        }
    }
};

const sass = {
    render(source, map, options) {
        return scss.render(source, map, Object.assign({}, options, {
            indentedSyntax: true
        }));
    }
};

const less = {
    render(source, map, options) {
        const nodeLess = customRequire("less", options);
        let result;
        let error = null;
        nodeLess.render(source, Object.assign({}, options, {
            syncImport: true
        }), ((err, output) => {
            error = err;
            result = output;
        }));
        if (error) return {
            code: "",
            errors: [ error ]
        };
        if (map) {
            return {
                code: result.css.toString(),
                map: merge(map, result.map),
                errors: []
            };
        }
        return {
            code: result.css.toString(),
            errors: []
        };
    }
};

const styl = {
    render(source, map, options) {
        const nodeStylus = customRequire("stylus", options);
        try {
            const ref = nodeStylus(source);
            Object.keys(options).forEach((key => ref.set(key, options[key])));
            if (map) ref.set("sourcemap", {
                inline: false,
                comment: false
            });
            const result = ref.render();
            if (map) {
                return {
                    code: result,
                    map: merge(map, ref.sourcemap),
                    errors: []
                };
            }
            return {
                code: result,
                errors: []
            };
        } catch (e) {
            return {
                code: "",
                errors: [ e ]
            };
        }
    }
};

const processors = {
    less: less,
    sass: sass,
    scss: scss,
    styl: styl,
    stylus: styl
};

const targetBrowserBabelPluginsHash = hash(...Object.keys(Object.assign({}, typeof ___targetBrowserBabelPlugins !== "undefined" ? ___targetBrowserBabelPlugins : {})));

const genSourcemap$1 = !!false;

const isProd = process.env.NODE_ENV === "production";

async function createSFCModule(source, filename, options) {
    var _a;
    const strFilename = filename.toString();
    const component = {};
    const {delimiters: delimiters, whitespace: whitespace, moduleCache: moduleCache, compiledCache: compiledCache, getResource: getResource, addStyle: addStyle, log: log, additionalBabelParserPlugins: additionalBabelParserPlugins = [], additionalBabelPlugins: additionalBabelPlugins = {}, customBlockHandler: customBlockHandler, devMode: devMode = false, createCJSModule: createCJSModule, processStyles: processStyles} = options;
    const descriptor = parse({
        source: source,
        filename: strFilename,
        needMap: genSourcemap$1,
        compiler: vueTemplateCompiler
    });
    const customBlockCallbacks = customBlockHandler !== undefined ? await Promise.all(descriptor.customBlocks.map((block => customBlockHandler(block, filename, options)))) : [];
    const scopeId = `data-v-${hash(strFilename)}`;
    if (descriptor.template && descriptor.template.lang) await loadModuleInternal({
        refPath: filename,
        relPath: descriptor.template.lang
    }, options);
    const hasScoped = descriptor.styles.some((e => e.scoped));
    if (hasScoped) {
        Object.assign(component, {
            _scopeId: scopeId
        });
    }
    const compileTemplateOptions = descriptor.template ? {
        source: descriptor.template.src ? await (await getResource({
            refPath: filename,
            relPath: descriptor.template.src
        }, options).getContent()).getContentData(false) : descriptor.template.content,
        filename: strFilename,
        compiler: vueTemplateCompiler,
        compilerOptions: {
            delimiters: delimiters,
            whitespace: whitespace,
            outputSourceRange: true,
            scopeId: hasScoped ? scopeId : null,
            comments: true
        },
        isProduction: isProd,
        prettify: devMode
    } : null;
    if ((_a = descriptor.template) === null || _a === void 0 ? void 0 : _a.lang) {
        const preprocess = moduleCache[descriptor.template.lang];
        compileTemplateOptions.source = await withCache(compiledCache, [ vueVersion, compileTemplateOptions.source, descriptor.template.lang ], (async ({preventCache: preventCache}) => await new Promise(((resolve, reject) => {
            preprocess.render(compileTemplateOptions.source, compileTemplateOptions.preprocessOptions, ((_err, _res) => {
                if (_err) reject(_err); else resolve(_res);
            }));
        }))));
    }
    if (descriptor.script) {
        const src = descriptor.script.src ? await (await getResource({
            refPath: filename,
            relPath: descriptor.script.src
        }, options).getContent()).getContentData(false) : descriptor.script.content;
        const [depsList, transformedScriptSource] = await withCache(compiledCache, [ vueVersion, isProd, devMode, src, descriptor.script.lang, additionalBabelParserPlugins, Object.keys(additionalBabelPlugins), targetBrowserBabelPluginsHash ], (async ({preventCache: preventCache}) => {
            var _a;
            let contextBabelParserPlugins = [ "jsx" ];
            let contextBabelPlugins = {
                jsx: jsx,
                babelSugarInjectH: babelSugarInjectH
            };
            if (((_a = descriptor.script) === null || _a === void 0 ? void 0 : _a.lang) === "ts") {
                contextBabelParserPlugins = [ ...contextBabelParserPlugins, "typescript" ];
                contextBabelPlugins = Object.assign(Object.assign({}, contextBabelPlugins), {
                    typescript: babelPlugin_typescript
                });
            }
            return await transformJSCode(src, true, strFilename, [ ...contextBabelParserPlugins, ...additionalBabelParserPlugins ], Object.assign(Object.assign({}, contextBabelPlugins), additionalBabelPlugins), log, devMode);
        }));
        await loadDeps(filename, depsList, options);
        Object.assign(component, interopRequireDefault(createCJSModule(filename, transformedScriptSource, options).exports).default);
    }
    if (descriptor.template !== null) {
        const [templateDepsList, templateTransformedSource] = await withCache(compiledCache, [ vueVersion, devMode, compileTemplateOptions.source, delimiters, whitespace, scopeId, additionalBabelParserPlugins, Object.keys(additionalBabelPlugins), targetBrowserBabelPluginsHash ], (async ({preventCache: preventCache}) => {
            const template = compileTemplate(compileTemplateOptions);
            template.code += `\nmodule.exports = { render: render, staticRenderFns: staticRenderFns }`;
            if (template.errors.length) {
                preventCache();
                for (let err of template.errors) {
                    if (typeof err !== "object") {
                        err = {
                            msg: err,
                            start: undefined,
                            end: undefined
                        };
                    }
                    log === null || log === void 0 ? void 0 : log("error", "SFC template", formatErrorStartEnd(err.msg, strFilename, compileTemplateOptions.source.trim(), err.start, err.end));
                }
            }
            for (let err of template.tips) {
                if (typeof err !== "object") {
                    err = {
                        msg: err,
                        start: undefined,
                        end: undefined
                    };
                }
                log === null || log === void 0 ? void 0 : log("info", "SFC template", formatErrorStartEnd(err.msg, strFilename, source, err.start, err.end));
            }
            return await transformJSCode(template.code, true, filename, additionalBabelParserPlugins, additionalBabelPlugins, log, devMode);
        }));
        await loadDeps(filename, templateDepsList, options);
        Object.assign(component, createCJSModule(filename, templateTransformedSource, options).exports);
    }
    for (const descStyle of descriptor.styles) {
        const srcRaw = descStyle.src ? await (await getResource({
            refPath: filename,
            relPath: descStyle.src
        }, options).getContent()).getContentData(false) : descStyle.content;
        const style = await withCache(compiledCache, [ vueVersion, srcRaw, descStyle.lang, scopeId, descStyle.scoped ], (async ({preventCache: preventCache}) => {
            const src = processStyles !== undefined ? await processStyles(srcRaw, descStyle.lang, filename, options) : srcRaw;
            if (src === undefined) preventCache();
            const compileStyleOptions = Object.assign({
                source: src,
                filename: strFilename,
                id: scopeId,
                scoped: descStyle.scoped !== undefined ? descStyle.scoped : false,
                trim: false
            }, processStyles === undefined ? {
                preprocessLang: descStyle.lang,
                preprocessOptions: {
                    preprocessOptions: {
                        customRequire: id => moduleCache[id]
                    }
                }
            } : {});
            if (descStyle.lang && processors[descStyle.lang] === undefined) processors[descStyle.lang] = await loadModuleInternal({
                refPath: filename,
                relPath: descStyle.lang
            }, options);
            const compiledStyle = await compileStyleAsync(compileStyleOptions);
            if (compiledStyle.errors.length) {
                preventCache();
                for (const err of compiledStyle.errors) {
                    log === null || log === void 0 ? void 0 : log("error", "SFC style", formatError(err, strFilename));
                }
            }
            return compiledStyle.code;
        }));
        addStyle(style, descStyle.scoped ? scopeId : undefined);
    }
    if (customBlockHandler !== undefined) await Promise.all(customBlockCallbacks.map((cb => cb === null || cb === void 0 ? void 0 : cb(component))));
    return component;
}

const genSourcemap = !!false;

const version$1 = "0.9.5";

function formatError(message, path, source) {
    return path + "\n" + message;
}

function formatErrorLineColumn(message, path, source, line, column) {
    if (!line) {
        return formatError(message, path);
    }
    const location = {
        start: {
            line: line,
            column: column
        }
    };
    return formatError(codeFrameColumns(source, location, {
        message: message
    }), path);
}

function formatErrorStartEnd(message, path, source, start, end) {
    if (!start) {
        return formatError(message, path);
    }
    const location = {
        start: {
            line: 1,
            column: start
        }
    };
    if (end) {
        location.end = {
            line: 1,
            column: end
        };
    }
    return formatError(codeFrameColumns(source, location, {
        message: message
    }), path);
}

function hash(...valueList) {
    return valueList.reduce(((hashInstance, val) => hashInstance.append(String(val))), new SparkMD5).end();
}

async function withCache(cacheInstance, key, valueFactory) {
    let cachePrevented = false;
    const api = {
        preventCache: () => cachePrevented = true
    };
    if (cacheInstance === undefined) return await valueFactory(api);
    const hashedKey = hash(...key);
    const valueStr = await cacheInstance.get(hashedKey);
    if (valueStr !== undefined) return JSON.parse(valueStr);
    const value = await valueFactory(api);
    if (cachePrevented === false) await cacheInstance.set(hashedKey, JSON.stringify(value));
    return value;
}

class Loading {
    constructor(promise) {
        this.promise = promise;
    }
}

function interopRequireDefault(obj) {
    return obj && obj.__esModule ? obj : {
        default: obj
    };
}

function renameDynamicImport(fileAst) {
    traverse(fileAst, {
        CallExpression(path) {
            if (types.isImport(path.node.callee)) path.replaceWith(types.callExpression(types.identifier("__vsfcl_import__"), path.node.arguments));
        }
    });
}

function parseDeps(fileAst) {
    const requireList = [];
    traverse(fileAst, {
        ExportAllDeclaration(path) {
            requireList.push(path.node.source.value);
        },
        ImportDeclaration(path) {
            requireList.push(path.node.source.value);
        },
        CallExpression(path) {
            if (path.node.callee.name === "require" && path.node.arguments.length === 1 && types.isStringLiteral(path.node.arguments[0])) {
                requireList.push(path.node.arguments[0].value);
            }
        }
    });
    return requireList;
}

const targetBrowserBabelPlugins = Object.assign({}, typeof ___targetBrowserBabelPlugins !== "undefined" ? ___targetBrowserBabelPlugins : {});

async function transformJSCode(source, moduleSourceType, filename, additionalBabelParserPlugins, additionalBabelPlugins, log, devMode = false) {
    let ast;
    try {
        ast = parse$1(source, {
            sourceType: moduleSourceType ? "module" : "script",
            sourceFilename: filename.toString(),
            plugins: [ ...additionalBabelParserPlugins !== undefined ? additionalBabelParserPlugins : [] ]
        });
    } catch (ex) {
        log === null || log === void 0 ? void 0 : log("error", "parse script", formatErrorLineColumn(ex.message, filename.toString(), source, ex.loc.line, ex.loc.column + 1));
        throw ex;
    }
    renameDynamicImport(ast);
    const depsList = parseDeps(ast);
    const transformedScript = await transformFromAstAsync(ast, source, {
        sourceMaps: genSourcemap,
        plugins: [ ...moduleSourceType ? [ babelPluginTransformModulesCommonjs ] : [], ...Object.values(targetBrowserBabelPlugins), ...additionalBabelPlugins !== undefined ? Object.values(additionalBabelPlugins) : [] ],
        babelrc: false,
        configFile: false,
        highlightCode: false,
        compact: !devMode,
        comments: devMode,
        retainLines: devMode,
        sourceType: moduleSourceType ? "module" : "script"
    });
    if (transformedScript === null || transformedScript.code == null) {
        const msg = `unable to transform script "${filename.toString()}"`;
        log === null || log === void 0 ? void 0 : log("error", msg);
        throw new Error(msg);
    }
    return [ depsList, transformedScript.code ];
}

async function loadModuleInternal(pathCx, options) {
    const {moduleCache: moduleCache, loadModule: loadModule, handleModule: handleModule} = options;
    const {id: id, path: path, getContent: getContent} = options.getResource(pathCx, options);
    if (id in moduleCache) {
        if (moduleCache[id] instanceof Loading) return await moduleCache[id].promise; else return moduleCache[id];
    }
    moduleCache[id] = new Loading((async () => {
        let module = undefined;
        if (loadModule) module = await loadModule(id, options);
        if (module === undefined) {
            const {getContentData: getContentData, type: type} = await getContent();
            if (handleModule !== undefined) module = await handleModule(type, getContentData, path, options);
            if (module === undefined) module = await handleModuleInternal(type, getContentData, path, options);
            if (module === undefined) throw new TypeError(`Unable to handle ${type} files (${path})`);
        }
        return moduleCache[id] = module;
    })());
    return await moduleCache[id].promise;
}

function defaultCreateCJSModule(refPath, source, options) {
    const {moduleCache: moduleCache, pathResolve: pathResolve, getResource: getResource} = options;
    const require = function(relPath) {
        const {id: id} = getResource({
            refPath: refPath,
            relPath: relPath
        }, options);
        if (id in moduleCache) return moduleCache[id];
        throw new Error(`require(${JSON.stringify(id)}) failed. module not found in moduleCache`);
    };
    const importFunction = async function(relPath) {
        return await loadModuleInternal({
            refPath: refPath,
            relPath: relPath
        }, options);
    };
    const module = {
        exports: {}
    };
    const moduleFunction = Function("exports", "require", "module", "__filename", "__dirname", "__vsfcl_import__", source);
    moduleFunction.call(module.exports, module.exports, require, module, refPath, pathResolve({
        refPath: refPath,
        relPath: "."
    }, options), importFunction);
    return module;
}

async function createJSModule(source, moduleSourceType, filename, options) {
    const {compiledCache: compiledCache, additionalBabelParserPlugins: additionalBabelParserPlugins, additionalBabelPlugins: additionalBabelPlugins, createCJSModule: createCJSModule, log: log} = options;
    const [depsList, transformedSource] = await withCache(compiledCache, [ version$1, source, filename, options.devMode, additionalBabelParserPlugins ? additionalBabelParserPlugins : "", additionalBabelPlugins ? Object.keys(additionalBabelPlugins) : "" ], (async () => await transformJSCode(source, moduleSourceType, filename, additionalBabelParserPlugins, additionalBabelPlugins, log, options.devMode)));
    await loadDeps(filename, depsList, options);
    return createCJSModule(filename, transformedSource, options).exports;
}

async function loadDeps(refPath, deps, options) {
    await Promise.all(deps.map((relPath => loadModuleInternal({
        refPath: refPath,
        relPath: relPath
    }, options))));
}

async function handleModuleInternal(type, getContentData, path, options) {
    var _a, _b;
    switch (type) {
      case ".vue":
        return createSFCModule(await getContentData(false), path, options);

      case ".js":
        return createJSModule(await getContentData(false), false, path, options);

      case ".mjs":
        return createJSModule(await getContentData(false), true, path, options);

      case ".ts":
        return createJSModule(await getContentData(false), true, path, Object.assign(Object.assign({}, options), {
            additionalBabelParserPlugins: [ "typescript", ...(_a = options.additionalBabelParserPlugins) !== null && _a !== void 0 ? _a : [] ],
            additionalBabelPlugins: Object.assign({
                typescript: babelPlugin_typescript
            }, (_b = options.additionalBabelPlugins) !== null && _b !== void 0 ? _b : {})
        }));
    }
    return undefined;
}

const version = "0.9.5";

const vueVersion = "2.7.16";

function throwNotDefined(details) {
    throw new ReferenceError(`${details} is not defined`);
}

const defaultGetPathname = path => {
    const searchPos = path.indexOf("?");
    if (searchPos !== -1) return path.slice(0, searchPos);
    return path;
};

const defaultPathResolve = ({refPath: refPath, relPath: relPath}, options) => {
    const {getPathname: getPathname} = options;
    if (refPath === undefined) return relPath;
    const relPathStr = relPath.toString();
    if (relPathStr[0] !== ".") return relPath;
    return posix.normalize(posix.join(posix.dirname(getPathname(refPath.toString())), relPathStr));
};

function defaultGetResource(pathCx, options) {
    const {pathResolve: pathResolve, getPathname: getPathname, getFile: getFile, log: log} = options;
    const path = pathResolve(pathCx, options);
    const pathStr = path.toString();
    return {
        id: pathStr,
        path: path,
        getContent: async () => {
            const res = await getFile(path);
            if (typeof res === "string" || res instanceof ArrayBuffer) {
                return {
                    type: posix.extname(getPathname(pathStr)),
                    getContentData: async asBinary => {
                        if (res instanceof ArrayBuffer !== asBinary) log === null || log === void 0 ? void 0 : log("warn", `unexpected data type. ${asBinary ? "binary" : "string"} is expected for "${path}"`);
                        return res;
                    }
                };
            }
            if (!res) {
                log === null || log === void 0 ? void 0 : log("error", `There is no file avaialable such as "${path}"`);
            }
            return {
                type: res.type !== undefined ? res.type : posix.extname(getPathname(pathStr)),
                getContentData: res.getContentData
            };
        }
    };
}

async function loadModule(path, options = throwNotDefined("options")) {
    var _a;
    const {moduleCache: moduleCache = throwNotDefined("options.moduleCache"), getFile: getFile = throwNotDefined("options.getFile()"), addStyle: addStyle = throwNotDefined("options.addStyle()"), pathResolve: pathResolve = defaultPathResolve, getResource: getResource = defaultGetResource, createCJSModule: createCJSModule = defaultCreateCJSModule, getPathname: getPathname = defaultGetPathname} = options;
    if (moduleCache instanceof Object) Object.setPrototypeOf(moduleCache, null);
    const normalizedOptions = Object.assign({
        moduleCache: moduleCache,
        pathResolve: pathResolve,
        getResource: getResource,
        createCJSModule: createCJSModule,
        getPathname: getPathname
    }, options);
    if (options.devMode) {
        if (options.compiledCache === undefined) (_a = options.log) === null || _a === void 0 ? void 0 : _a.call(options, "info", "options.compiledCache is not defined, performance will be affected");
    }
    return await loadModuleInternal({
        refPath: undefined,
        relPath: path
    }, normalizedOptions);
}

function buildTemplateProcessor(processor) {
    return {
        render: (source, preprocessOptions, cb) => {
            try {
                const ret = processor(source, preprocessOptions);
                if (typeof ret === "string") {
                    cb(null, ret);
                } else {
                    ret.then((processed => {
                        cb(null, processed);
                    }));
                    ret.catch((err => {
                        cb(err, null);
                    }));
                }
            } catch (err) {
                cb(err, null);
            }
        }
    };
}

export { buildTemplateProcessor, loadModule, version, vueVersion };
