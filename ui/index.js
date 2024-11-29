; (async () => {

  const { loadModule, version } = window["vue3-sfc-loader"];
  console.info("version of vue3-sfc-loader:",version);

  //vue create
  const options = {

    moduleCache: {
      vue: Vue,
      vueRouter: VueRouter,
      request:request,
      config:config,
      //apis
      sseApi,
      scheduleApi,
      eventApi,
      loginApi,
      userApi,
      dlqApi,
      //apis end
    },

    async getFile(url) {
      const res = await fetch(url);

      if ( !res.ok )
        throw Object.assign(new Error(res.statusText + ' ' + url), { res });
      return {
        getContentData: (asBinary ) => asBinary ? res.arrayBuffer() : res.text(),
        type:".vue"
      }
    },

    addStyle(styleStr) {
      const style = document.createElement('style');
      style.textContent = styleStr;
      const ref = document.head.getElementsByTagName('style')[0] || null;
      document.head.insertBefore(style, ref);
    },
    log(type, ...args) {
      console.log(type, ...args);
    }
  }
  // admin routes
  const adminRoute =  {
        path:'/admin',component:()=>loadModule("./src/layout/adminMain.vue",options),
        children:[
          { path: '', component: () => loadModule('./src/pages/home.vue', options) },
          { path: 'home',component:()=>loadModule("./src/pages/home.vue",options)},
          //{ path: 'server', component: () => loadModule('./src/pages/server.vue', options) },
          { path: 'schedule', component: () => loadModule("./src/pages/schedule.vue", options) },
          { path: 'queue', component: () => loadModule("./src/pages/queue/list.vue", options) },
          { path: 'queue/detail/:id',component:()=>loadModule("./src/pages/queue/detail.vue",options)},
          { path: 'log/event',component:()=>loadModule("./src/pages/log/event.vue",options)},
          { path: 'log/workflow',component:()=>loadModule("./src/pages/log/workflow.vue",options)},
          { path: 'log/dlq',component:()=>loadModule("./src/pages/log/dlq.vue",options)},
          // { path: 'log/success',component:()=>loadModule("./src/pages/log/success.vue",options)},
          // { path: 'log/error',component:()=>loadModule("./src/pages/log/error.vue",options)},
          { path: 'log/detail/:id',component:()=>loadModule("./src/pages/log/detail.vue",options)},
          { path: 'redis', component: () => loadModule("./src/pages/redis/info.vue", options) },
          {path:'redis/monitor',component:()=>loadModule("./src/pages/redis/monitor.vue",options)},
          { path: 'user',component:()=>loadModule("./src/pages/user.vue",options)}
        ]
  };
  // login route
  const loginRoute = { path:"/login",component:()=>loadModule("./src/pages/login.vue",options)};

  // router
  const router = VueRouter.createRouter({
    history: VueRouter.createWebHashHistory(),
    routes: [
      {path:'/',redirect:'/admin'},
      adminRoute,
      loginRoute
    ],
  });
  router.beforeEach((to, from) => {
    let token = sessionStorage.getItem("token");
    if (token == null && to.path !== "/login"){
      return {path:"/login"};
    }
  })

  const app = Vue.createApp({
    components: {
      'mainLayout': Vue.defineAsyncComponent(() => loadModule('./src/layout/main.vue', options)),
    },
    template: `<mainLayout/>`
  });
  app.component("v-chart",VueECharts);
  app.use(router);
  app.mount('#app');

})().catch(ex => console.log(ex))