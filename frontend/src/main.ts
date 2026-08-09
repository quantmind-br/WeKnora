import { createApp } from "vue";
import { createPinia } from "pinia";
import App from "./App.vue";
import router from "./router";
import "./assets/fonts.css";
import TDesign from "tdesign-vue-next";
// Import a small set of global style variables from the component library
import "tdesign-vue-next/dist/tdesign.css";
import "@/assets/theme/theme.css";
import "@/assets/dropdown-menu.less";
import "@/components/css/chat-hljs-dark.less";
// vue-virtual-scroller ships its own tiny stylesheet — required for
// RecycleScroller/DynamicScroller to size their viewport correctly.
// Without it the scroller computes 0 height and renders no items.
import "vue-virtual-scroller/dist/vue-virtual-scroller.css";
import i18n from "./i18n";
import { initTheme } from "@/composables/useTheme";
import { initFont } from "@/composables/useFont";
import { installTDesignIconOfflineGuard } from "@/utils/tdesign-icon-offline";
import { installAutofillGuard } from "@/utils/disable-autofill";

// Must run before the Vue component mounts, to prevent tdesign-icons from making a runtime request to tdesign.gtimg.com
installTDesignIconOfflineGuard();

initTheme();
initFont();

const app = createApp(App);

// Global error handling: catch unhandled component errors to prevent a blank/white screen
app.config.errorHandler = (err, instance, info) => {
  console.error("[WeKnora] Unhandled Vue error:", err, "\nComponent:", instance, "\nInfo:", info);
};

app.use(TDesign);
app.use(createPinia());
app.use(router);
app.use(i18n);

// Mount only after the first-screen route (including navigation guards and Lite auto-login) completes, to avoid flashing the default page before redirecting
router.isReady().finally(() => {
  app.mount("#app");
  installAutofillGuard();
});
