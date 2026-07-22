; (async () => {

  const { loadModule, version } = window["vue3-sfc-loader"];
  const compiledCache = createCompiledCache(`beanq:sfc:${Vue.version}:${version}:`);
  const i18n = VueI18n.createI18n({
    legacy:false,
    locale:"ja",
  });

  const options = {

    compiledCache,

    moduleCache: {
      vue: Vue,
      vueRouter: VueRouter,
      request:request,
      config:config,
      i18n:i18n,
      Base,
      //apis
      sseApi,
      scheduleApi,
      eventApi,
      loginApi,
      userApi,
      dlqApi,
      dashboardApi,
      logApi,
	  roleApi
      //apis end
    },

    async getFile(url) {

      const resourceURL = resolveLoaderURL(url);
      const res = await fetch(resourceURL, {cache: "no-cache", credentials: "same-origin"});

      if ( !res.ok ){
		if (res.status === 403 && typeof redirectToLogin === "function") {
			redirectToLogin();
		}
        throw Object.assign(new Error(res.statusText + ' ' + url), { res });
      }
      return res.text();

    },

    addStyle(styleStr) {
      const style = document.createElement('style');
      style.textContent = styleStr;
      const ref = document.head.getElementsByTagName('style')[0] || null;
      document.head.insertBefore(style, ref);
    },
    customBlockHandler(block, filename, options){

      if ( block.type !== 'i18n' )
        return

      const messages = JSON.parse(block.content);

      for ( let locale in messages ){
        i18n.global.mergeLocaleMessage(locale, messages[locale]);
      }
    },
    log(type, ...args) {
      console.log(type, ...args);
    }
  }
  // admin routes
  const adminRoute =  {
        path:'/admin',component:()=>loadModule("./src/layout/adminMain.vue",options),
        children:[
          { path: 'home',component:()=>loadModule("./src/pages/home.vue",options)},
          { path: 'schedule', component: () => loadModule("./src/pages/schedule.vue", options) },
          { path: 'queue', component: () => loadModule("./src/pages/queue/list.vue", options) },
          { path: 'queue/detail/:id',component:()=>loadModule("./src/pages/queue/detail.vue",options)},
          { path: 'log/event',component:()=>loadModule("./src/pages/log/event/event.vue",options)},
          { path: 'log/detail/:id',component:()=>loadModule("./src/pages/log/event/detail.vue",options)},
          { path: 'log/dlq',component:()=>loadModule("./src/pages/log/dlq/dlq.vue",options)},
          { path: 'log/dlq/detail/:id',component:()=>loadModule("./src/pages/log/dlq/detail.vue",options)},
          { path: 'log/workflow',component:()=>loadModule("./src/pages/log/workflow/workflow.vue",options)},
          { path: 'redis', component: () => loadModule("./src/pages/redis/info.vue", options) },
          { path: 'redis/monitor',component:()=>loadModule("./src/pages/redis/monitor.vue",options)},
          { path: 'user',component:()=>loadModule("./src/pages/user/user.vue",options)},
          { path: 'optLog',component:()=>loadModule("./src/pages/setting/optLog.vue",options)},
          { path: 'role',component:()=>loadModule("./src/pages/setting/role.vue",options)},
          { path: 'db-size',component:()=>loadModule("./src/pages/redis/dbsize.vue",options)},
          { path: 'config',component:()=>loadModule("./src/pages/setting/config.vue",options)},
          { path: 'tenant',component:()=>loadModule("./src/pages/tenant/list.vue",options)},
          { path: 'tenant/add',component:()=>loadModule("./src/pages/tenant/add.vue",options)},
          { path: 'database',component:()=>loadModule("./src/pages/setting/database.vue",options)}

        ]
  };

  // login route
  const loginRoute = { path:"/login",component:()=>loadModule("./src/pages/login.vue",options)};

  // router
  const router = VueRouter.createRouter({
    history: VueRouter.createWebHashHistory(),
    routes: [
      {path:'/',redirect:'/admin/home'},
      adminRoute,
      loginRoute
    ],
  });
  router.beforeEach((to, from) => {
	let authenticated = Storage.GetItem("authenticated");
	if (authenticated == null && to.path !== "/login"){
      return {path:"/login",replace:true};
    }
  })

  const app = Vue.createApp({
    components: {
      'mainLayout': Vue.defineAsyncComponent(() => loadModule('./src/layout/main.vue', options)),
    },
    template: `<mainLayout/>`
  });
  app.component("v-chart",VueECharts);
  app.component("vue-date-picker",VueDatePicker );
  app.use(router);
  app.use(i18n);
  app.mount('#app');

  if (Storage.GetItem("authenticated") != null) {
	const preload = () => Promise.allSettled([
		loadModule("./src/layout/adminMain.vue", options),
		loadModule("./src/pages/home.vue", options),
	]);
	if (typeof window.requestIdleCallback === "function") {
		window.requestIdleCallback(preload);
	} else {
		window.setTimeout(preload, 0);
	}
  }

})().catch(ex => console.log(ex))

function resolveLoaderURL(resource) {
	const url = new URL(resource, window.location.href);
	const allowedPath = url.pathname.startsWith("/src/") || url.pathname.startsWith("/static/");
	if (url.origin !== window.location.origin || !allowedPath || url.pathname.includes("..")) {
		throw new TypeError(`Unsupported UI resource: ${resource}`);
	}
	return url;
}

function createCompiledCache(namespace) {
	const memory = new Map();
	return {
		async get(key) {
			const storageKey = namespace + await digestCacheKey(key);
			if (memory.has(storageKey)) {
				return memory.get(storageKey);
			}
			try {
				const value = localStorage.getItem(storageKey);
				if (value !== null) {
					memory.set(storageKey, value);
				}
				return value ?? undefined;
			} catch {
				return undefined;
			}
		},
		async set(key, value) {
			const storageKey = namespace + await digestCacheKey(key);
			memory.set(storageKey, value);
			try {
				localStorage.setItem(storageKey, value);
			} catch {
				removeCompiledCache("beanq:sfc:");
				try {
					localStorage.setItem(storageKey, value);
				} catch {}
			}
		},
	};
}

async function digestCacheKey(value) {
	if (globalThis.crypto?.subtle && globalThis.TextEncoder) {
		const bytes = new TextEncoder().encode(value);
		const digest = await crypto.subtle.digest("SHA-256", bytes);
		return Array.from(new Uint8Array(digest), byte => byte.toString(16).padStart(2, "0")).join("");
	}
	let hash = 2166136261;
	for (let index = 0; index < value.length; index++) {
		hash ^= value.charCodeAt(index);
		hash = Math.imul(hash, 16777619);
	}
	return (hash >>> 0).toString(16);
}

function removeCompiledCache(namespace) {
	try {
		for (let index = localStorage.length - 1; index >= 0; index--) {
			const key = localStorage.key(index);
			if (key?.startsWith(namespace)) {
				localStorage.removeItem(key);
			}
		}
	} catch {}
}
