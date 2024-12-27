import { ModuleExport, Options, LangProcessor, AbstractPath } from './types';
export * from './types';
/**
 * the version of the library (process.env.VERSION is set by webpack, at compile-time)
 */
export declare const version: string;
/**
 * the version of Vue that is expected by the library
 */
export { vueVersion } from './createSFCModule';
export { createCJSModule } from './tools';
/**
 * This is the main function.
 * This function is intended to be used only to load the entry point of your application.
 * If for some reason you need to use it in your components, be sure to share at least the options.`compiledCache` object between all calls.
 *
 * @param path  The path of the `.vue` file. If path is not a path (eg. an string ID), your [[getFile]] function must return a [[File]] object.
 * @param options  The options
 * @returns A Promise of the component
 *
 * **example using `Vue.defineAsyncComponent`:**
 *
 * ```javascript
 *
 *	const app = Vue.createApp({
 *		components: {
 *			'my-component': Vue.defineAsyncComponent( () => loadModule('./myComponent.vue', options) )
 *		},
 *		template: '<my-component></my-component>'
 *	});
 *
 * ```
 *
 * **example using `await`:**
 *
 * ```javascript

 * ;(async () => {
 *
 *		const app = Vue.createApp({
 *			components: {
 *				'my-component': await loadModule('./myComponent.vue', options)
 *			},
 *			template: '<my-component></my-component>'
 *		});
 *
 * })()
 * .catch(ex => console.error(ex));
 *
 * ```
 *
 */
export declare function loadModule(path: AbstractPath, options?: Options): Promise<ModuleExport>;
/**
 * Convert a function to template processor interface (consolidate)
 */
export declare function buildTemplateProcessor(processor: LangProcessor): {
    render: (source: string, preprocessOptions: string, cb: (_err: any, _res: any) => void) => void;
};
