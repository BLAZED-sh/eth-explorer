import { createApp } from "vue";
import App from "./App.vue";
import { router } from "./routes";
import { connect } from "./ws";
import "./style.css";

connect();
createApp(App).use(router).mount("#app");
