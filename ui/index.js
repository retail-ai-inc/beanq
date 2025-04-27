; (async () => {

  const { loadModule, version } = window["vue3-sfc-loader"];
  const i18n = VueI18n.createI18n({
    legacy:false,
    locale:"ja",
  });

  const options = {

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
      roleApi,
      //apis end
    },

    async getFile(url) {

      const headers = new Headers();
      headers.set("Cache-Control",'no-cache, no-store, must-revalidate');
      headers.set("Pragma",'no-cache');
      headers.set("Expires",'0');

      const res = await fetch(url,{
        cache: 'no-store',
        headers: {
          ...headers,
        }
      });

      if ( !res.ok ){
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
          { path: 'log/workflow',component:()=>loadModule("./src/pages/log/workflow/workflow.vue",options)},
          { path: 'log/dlq',component:()=>loadModule("./src/pages/log/dlq/dlq.vue",options)},
          { path: 'log/dlq/detail/:id',component:()=>loadModule("./src/pages/log/dlq/detail.vue",options)},
          {path: 'log/workflow',component:()=>loadModule("./src/pages/log/workflow/workflow.vue",options)},
          { path: 'redis', component: () => loadModule("./src/pages/redis/info.vue", options) },
          { path:'redis/monitor',component:()=>loadModule("./src/pages/redis/monitor.vue",options)},
          { path: 'user',component:()=>loadModule("./src/pages/user/user.vue",options)},
          { path:'optLog',component:()=>loadModule("./src/pages/setting/optLog.vue",options)},
          { path:'role',component:()=>loadModule("./src/pages/setting/role.vue",options)},
          {path: 'db-size',component:()=>loadModule("./src/pages/redis/dbsize.vue",options)},
          {path: 'config',component:()=>loadModule("./src/pages/setting/config.vue",options)}
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
    let token = Storage.GetItem("token");
    if (token == null && to.path !== "/login"){
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

})().catch(ex => console.log(ex))