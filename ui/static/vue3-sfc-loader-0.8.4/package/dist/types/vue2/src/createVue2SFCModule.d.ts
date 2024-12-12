import { Options, ModuleExport, AbstractPath } from './types';
export { version as vueVersion } from 'vue-template-compiler/package.json';
/**
 * @internal
 */
export declare function createSFCModule(source: string, filename: AbstractPath, options: Options): Promise<ModuleExport>;
