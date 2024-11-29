import { types as t } from '@babel/core';
import { Cache, Options, ValueFactory, ModuleExport, Module, PathContext, AbstractPath } from './types';
/**
 * @internal
 */
export declare function formatError(message: string, path: string, source: string): string;
/**
 * @internal
 */
export declare function formatErrorLineColumn(message: string, path: string, source: string, line?: number, column?: number): string;
/**
 * @internal
 */
export declare function formatErrorStartEnd(message: string, path: string, source: string, start: number, end?: number): string;
/**
 * @internal
 */
export declare function hash(...valueList: any[]): string;
/**
 * Simple cache helper
 * preventCache usage: non-fatal error
 * @internal
 */
export declare function withCache(cacheInstance: Cache, key: any[], valueFactory: ValueFactory): Promise<any>;
/**
 * @internal
 */
export declare class Loading {
    promise: Promise<ModuleExport>;
    constructor(promise: Promise<ModuleExport>);
}
/**
 * @internal
 */
export declare function interopRequireDefault(obj: any): any;
/**
 * import is a reserved keyword, then rename
 * @internal
 */
export declare function renameDynamicImport(fileAst: t.File): void;
/**
 * @internal
 */
export declare function parseDeps(fileAst: t.File): string[];
/**
 * @internal
 */
export declare function transformJSCode(source: string, moduleSourceType: boolean, filename: AbstractPath, additionalBabelParserPlugins: Options['additionalBabelParserPlugins'], additionalBabelPlugins: Options['additionalBabelPlugins'], log: Options['log']): Promise<[string[], string]>;
export declare function loadModuleInternal(pathCx: PathContext, options: Options): Promise<ModuleExport>;
/**
 * Create a cjs module
 * @internal
 */
export declare function createCJSModule(refPath: AbstractPath, source: string, options: Options): Module;
/**
 * @internal
 */
export declare function createJSModule(source: string, moduleSourceType: boolean, filename: AbstractPath, options: Options): Promise<ModuleExport>;
/**
 * Just load and cache given dependencies.
 * @internal
 */
export declare function loadDeps(refPath: AbstractPath, deps: AbstractPath[], options: Options): Promise<void>;
