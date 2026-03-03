import { createRouter, createWebHashHistory } from "vue-router";
import LoginView from "../views/LoginView.vue";
import StarsView from "../views/StarsView.vue";
import SettingsView from "../views/SettingsView.vue";
import { getSessionToken } from "../services/sessionStorage";
import { useAuthStore } from "../stores/authStore";
import { useStarsStore } from "../stores/starsStore";

const protectedRouteNames = new Set(["stars", "settings"]);

export const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: "/", name: "login", component: LoginView },
    { path: "/stars", name: "stars", component: StarsView },
    { path: "/settings", name: "settings", component: SettingsView },
  ],
});

router.beforeEach(async (to) => {
  const token = await getSessionToken();
  const hasSession = token.length > 0;

  if (to.name && protectedRouteNames.has(String(to.name)) && !hasSession) {
    return { name: "login" };
  }

  if (to.name === "login" && hasSession) {
    return { name: "stars" };
  }

  return true;
});

router.afterEach((to) => {
  if (to.name !== "stars") {
    return;
  }

  const authStore = useAuthStore();
  if (!authStore.isAuthenticated) {
    authStore.initializeSession().catch(() => {
      // state handled in store
    });
  }

  const starsStore = useStarsStore();
  if (!starsStore.tags.length) {
    starsStore.fetchTags().catch(() => {
      // state handled in store
    });
  }
  if (!starsStore.syncJob) {
    starsStore.fetchSyncStatus().catch(() => {
      // state handled in store
    });
  }
});
