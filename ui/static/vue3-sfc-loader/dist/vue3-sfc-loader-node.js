"use strict";

var path = require("path");

var core = require("@babel/core");

var parser = require("@babel/parser");

var codeFrame = require("@babel/code-frame");

var babelPluginTransformModulesCommonjs = require("@babel/plugin-transform-modules-commonjs");

var babelPlugin_typescript = require("@babel/plugin-transform-typescript");

var SparkMD5 = require("spark-md5");

var compilerSfc = require("@vue/compiler-sfc");

var vue_CompilerDOM = require("@vue/compiler-dom");

var babelPlugin_jsx = require("@vue/babel-plugin-jsx");

function _interopNamespaceDefault(e) {
    var n = Object.create(null);
    if (e) {
        Object.keys(e).forEach((function(k) {
            if (k !== "default") {
                var d = Object.getOwnPropertyDescriptor(e, k);
                Object.defineProperty(n, k, d.get ? d : {
                    enumerable: true,
                    get: function() {
                        return e[k];
                    }
                });
            }
        }));
    }
    n.default = e;
    return Object.freeze(n);
}

var vue_CompilerDOM__namespace = _interopNamespaceDefault(vue_CompilerDOM);

const targetBrowserBabelPluginsHash = hash(...Object.keys(Object.assign({}, typeof ___targetBrowserBabelPlugins !== "undefined" ? ___targetBrowserBabelPlugins : {})));

const genSourcemap$1 = !!false;

const isProd = process.env.NODE_ENV === "production";

async function createSFCModule(source, filename, options) {
    var _a, _b, _c, _d, _e;
    const strFilename = filename.toString();
    const component = {};
    const {delimiters: delimiters, whitespace: whitespace, isCustomElement: isCustomElement, moduleCache: moduleCache, compiledCache: compiledCache, getResource: getResource, addStyle: addStyle, log: log, additionalBabelParserPlugins: additionalBabelParserPlugins = [], additionalBabelPlugins: additionalBabelPlugins = {}, customBlockHandler: customBlockHandler, devMode: devMode = false, createCJSModule: createCJSModule, processStyles: processStyles} = options;
    const {descriptor: descriptor, errors: errors} = compilerSfc.parse(source, {
        filename: strFilename,
        sourceMap: genSourcemap$1
    });
    const customBlockCallbacks = customBlockHandler !== undefined ? await Promise.all(descriptor.customBlocks.map((block => customBlockHandler(block, filename, options)))) : [];
    const scopeId = `data-v-${hash(strFilename)}`;
    const hasScoped = descriptor.styles.some((e => e.scoped));
    if (hasScoped) {
        component.__scopeId = scopeId;
    }
    if (descriptor.template && descriptor.template.lang) await loadModuleInternal({
        refPath: filename,
        relPath: descriptor.template.lang
    }, options);
    const compileTemplateOptions = descriptor.template ? {
        compiler: Object.assign(Object.assign({}, vue_CompilerDOM__namespace), {
            compile: (template, options) => vue_CompilerDOM__namespace.compile(template, Object.assign(Object.assign({}, options), {
                sourceMap: genSourcemap$1
            }))
        }),
        source: descriptor.template.src ? await (await getResource({
            refPath: filename,
            relPath: descriptor.template.src
        }, options).getContent()).getContentData(false) : descriptor.template.content,
        filename: descriptor.filename,
        isProd: isProd,
        scoped: hasScoped,
        id: scopeId,
        slotted: descriptor.slotted,
        compilerOptions: {
            isCustomElement: isCustomElement,
            whitespace: whitespace,
            delimiters: delimiters,
            scopeId: hasScoped ? scopeId : undefined,
            mode: "module"
        },
        preprocessLang: descriptor.template.lang,
        preprocessCustomRequire: id => moduleCache[id]
    } : undefined;
    if (descriptor.script || descriptor.scriptSetup) {
        if ((_a = descriptor.script) === null || _a === void 0 ? void 0 : _a.src) descriptor.script.content = await (await getResource({
            refPath: filename,
            relPath: descriptor.script.src
        }, options).getContent()).getContentData(false);
        const [bindingMetadata, depsList, transformedScriptSource] = await withCache(compiledCache, [ vueVersion, isProd, devMode, (_b = descriptor.script) === null || _b === void 0 ? void 0 : _b.content, (_c = descriptor.script) === null || _c === void 0 ? void 0 : _c.lang, (_d = descriptor.scriptSetup) === null || _d === void 0 ? void 0 : _d.content, (_e = descriptor.scriptSetup) === null || _e === void 0 ? void 0 : _e.lang, additionalBabelParserPlugins, Object.keys(additionalBabelPlugins), targetBrowserBabelPluginsHash ], (async ({preventCache: preventCache}) => {
            var _a, _b;
            let contextBabelParserPlugins = [ "jsx" ];
            let contextBabelPlugins = {
                jsx: babelPlugin_jsx
            };
            if (((_a = descriptor.script) === null || _a === void 0 ? void 0 : _a.lang) === "ts" || ((_b = descriptor.scriptSetup) === null || _b === void 0 ? void 0 : _b.lang) === "ts") {
                contextBabelParserPlugins = [ ...contextBabelParserPlugins, "typescript" ];
                contextBabelPlugins = Object.assign(Object.assign({}, contextBabelPlugins), {
                    typescript: babelPlugin_typescript
                });
            }
            const scriptBlock = compilerSfc.compileScript(descriptor, {
                isProd: isProd,
                sourceMap: genSourcemap$1,
                id: scopeId,
                babelParserPlugins: [ ...contextBabelParserPlugins, ...additionalBabelParserPlugins ],
                inlineTemplate: false,
                templateOptions: compileTemplateOptions
            });
            return [ scriptBlock.bindings, ...await transformJSCode(scriptBlock.content, true, strFilename, [ ...contextBabelParserPlugins, ...additionalBabelParserPlugins ], Object.assign(Object.assign({}, contextBabelPlugins), additionalBabelPlugins), log, devMode) ];
        }));
        if ((compileTemplateOptions === null || compileTemplateOptions === void 0 ? void 0 : compileTemplateOptions.compilerOptions) !== undefined) compileTemplateOptions.compilerOptions.bindingMetadata = bindingMetadata;
        await loadDeps(filename, depsList, options);
        Object.assign(component, interopRequireDefault(createCJSModule(filename, transformedScriptSource, options).exports).default);
    }
    if (descriptor.template !== null) {
        const [templateDepsList, templateTransformedSource] = await withCache(compiledCache, [ vueVersion, devMode, compileTemplateOptions.source, compileTemplateOptions.compilerOptions.delimiters, compileTemplateOptions.compilerOptions.whitespace, compileTemplateOptions.compilerOptions.scopeId, compileTemplateOptions.compilerOptions.bindingMetadata ? Object.entries(compileTemplateOptions.compilerOptions.bindingMetadata) : "", additionalBabelParserPlugins, Object.keys(additionalBabelPlugins), targetBrowserBabelPluginsHash ], (async ({preventCache: preventCache}) => {
            const template = compilerSfc.compileTemplate(compileTemplateOptions);
            if (template.errors.length) {
                preventCache();
                for (const err of template.errors) {
                    if (typeof err === "object") {
                        if (err.loc) {
                            log === null || log === void 0 ? void 0 : log("error", "SFC template", formatErrorLineColumn(err.message, strFilename, source, err.loc.start.line + descriptor.template.loc.start.line - 1, err.loc.start.column));
                        } else {
                            log === null || log === void 0 ? void 0 : log("error", "SFC template", formatError(err.message, strFilename));
                        }
                    } else {
                        log === null || log === void 0 ? void 0 : log("error", "SFC template", formatError(err, strFilename));
                    }
                }
            }
            for (const err of template.tips) log === null || log === void 0 ? void 0 : log("info", "SFC template", err);
            return await transformJSCode(template.code, true, descriptor.filename, additionalBabelParserPlugins, additionalBabelPlugins, log, devMode);
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
            if (processStyles === undefined && descStyle.lang !== undefined) await loadModuleInternal({
                refPath: filename,
                relPath: descStyle.lang
            }, options);
            const compiledStyle = await compilerSfc.compileStyleAsync(Object.assign({
                filename: descriptor.filename,
                source: src,
                isProd: isProd,
                id: scopeId,
                scoped: descStyle.scoped,
                trim: true
            }, processStyles === undefined ? {
                preprocessLang: descStyle.lang,
                preprocessCustomRequire: id => moduleCache[id]
            } : {}));
            if (compiledStyle.errors.length) {
                preventCache();
                for (const err of compiledStyle.errors) {
                    log === null || log === void 0 ? void 0 : log("error", "SFC style", formatErrorLineColumn(err.message, filename, source, err.line + descStyle.loc.start.line - 1, err.column));
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
    return formatError(codeFrame.codeFrameColumns(source, location, {
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
    core.traverse(fileAst, {
        CallExpression(path) {
            if (core.types.isImport(path.node.callee)) path.replaceWith(core.types.callExpression(core.types.identifier("__vsfcl_import__"), path.node.arguments));
        }
    });
}

function parseDeps(fileAst) {
    const requireList = [];
    core.traverse(fileAst, {
        ExportAllDeclaration(path) {
            requireList.push(path.node.source.value);
        },
        ImportDeclaration(path) {
            requireList.push(path.node.source.value);
        },
        CallExpression(path) {
            if (path.node.callee.name === "require" && path.node.arguments.length === 1 && core.types.isStringLiteral(path.node.arguments[0])) {
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
        ast = parser.parse(source, {
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
    const transformedScript = await core.transformFromAstAsync(ast, source, {
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

const vueVersion = "3.4.15";

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
    return path.posix.normalize(path.posix.join(path.posix.dirname(getPathname(refPath.toString())), relPathStr));
};

function defaultGetResource(pathCx, options) {
    const {pathResolve: pathResolve, getPathname: getPathname, getFile: getFile, log: log} = options;
    const path$1 = pathResolve(pathCx, options);
    const pathStr = path$1.toString();
    return {
        id: pathStr,
        path: path$1,
        getContent: async () => {
            const res = await getFile(path$1);
            if (typeof res === "string" || res instanceof ArrayBuffer) {
                return {
                    type: path.posix.extname(getPathname(pathStr)),
                    getContentData: async asBinary => {
                        if (res instanceof ArrayBuffer !== asBinary) log === null || log === void 0 ? void 0 : log("warn", `unexpected data type. ${asBinary ? "binary" : "string"} is expected for "${path$1}"`);
                        return res;
                    }
                };
            }
            if (!res) {
                log === null || log === void 0 ? void 0 : log("error", `There is no file avaialable such as "${path$1}"`);
            }
            return {
                type: res.type !== undefined ? res.type : path.posix.extname(getPathname(pathStr)),
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

exports.buildTemplateProcessor = buildTemplateProcessor;

exports.loadModule = loadModule;

exports.version = version;

exports.vueVersion = vueVersion;
