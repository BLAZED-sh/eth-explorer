import { createRouter, createWebHistory } from "vue-router";

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/", name: "dashboard", component: () => import("./Dashboard.vue") },
    { path: "/tx/:hash", name: "tx", component: () => import("./Transaction.vue") },
    { path: "/blocks", name: "blocks", component: () => import("./Blocks.vue") },
    { path: "/block/:id", name: "block", component: () => import("./Block.vue") },
    { path: "/address/:address", name: "address", component: () => import("./Address.vue") },
    { path: "/:pathMatch(.*)*", name: "notfound", component: () => import("./NotFound.vue") },
  ],
  scrollBehavior() {
    return { top: 0 };
  },
});
