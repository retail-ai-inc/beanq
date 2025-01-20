import { Options, ModuleExport, AbstractPath } from './types';
/**
 * @internal
 */
export declare function createSFCModule(source: string, filename: AbstractPath, options: Options): Promise<ModuleExport>;
