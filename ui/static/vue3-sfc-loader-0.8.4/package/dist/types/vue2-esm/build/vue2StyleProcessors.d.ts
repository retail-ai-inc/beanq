/**
 * This is customized version of styleProcessors defined in @vue/component-compiler-utils to bring customRequire
 * support.
 *
 * https://github.com/vuejs/component-compiler-utils/blob/master/lib/styleProcessors/index.ts
 */
export interface StylePreprocessor {
    render(source: string, map: any | null, options: any): StylePreprocessorResults;
}
export interface StylePreprocessorResults {
    code: string;
    map?: any;
    errors: Array<Error>;
}
export declare const processors: {
    [key: string]: StylePreprocessor;
};
