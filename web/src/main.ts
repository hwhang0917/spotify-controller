import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import '@fontsource/geist-sans/400.css'
import '@fontsource/geist-sans/500.css'
import '@fontsource/geist-sans/600.css'
import '@fontsource/geist-mono/500.css'
import 'vue-sonner/style.css'
import './style.css'
import App from './App.vue'
import Home from './Home.vue'
import SearchPage from './SearchPage.vue'
import ArtistPage from './ArtistPage.vue'
import AlbumPage from './AlbumPage.vue'
import FolderPage from './FolderPage.vue'

const router = createRouter({
  history: createWebHistory(),
  scrollBehavior: () => ({ top: 0 }),
  routes: [
    { path: '/', component: Home },
    { path: '/search', component: SearchPage },
    { path: '/artist/:source/:id', component: ArtistPage, props: true },
    { path: '/album/:source/:id', component: AlbumPage, props: true },
    { path: '/folder/:source/:id?', component: FolderPage, props: true },
    // /join?invitationCode=… and anything unknown land on the home page; the
    // query is kept so App.vue can redeem the code.
    { path: '/:pathMatch(.*)*', redirect: (to) => ({ path: '/', query: to.query }) },
  ],
})

createApp(App).use(router).mount('#app')
